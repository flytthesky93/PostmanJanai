package service

import (
	"PostmanJanai/internal/entity"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// importInsomnia parses an Insomnia v4 export (JSON) into an ImportedCollection.
// Structure: { "_type":"export", "__export_format":4, "resources":[...] }
// Resources of interest:
//   workspace       → root scope (we use its name if the export has only one workspace)
//   request_group   → folder (nested via parentId chain)
//   request         → saved request under its parent folder / workspace
//   environment     → environment variables (first one seeded)
func importInsomnia(raw []byte) (*entity.ImportedCollection, error) {
	var doc struct {
		Type      string             `json:"_type"`
		Format    int                `json:"__export_format"`
		Resources []insomniaResource `json:"resources"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse insomnia json: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(doc.Type)) != "export" || len(doc.Resources) == 0 {
		return nil, errors.New("document is not an insomnia export")
	}

	// Index resources by _id for parent resolution and by _type for traversal.
	byID := make(map[string]*insomniaResource, len(doc.Resources))
	for i := range doc.Resources {
		r := &doc.Resources[i]
		byID[r.ID] = r
	}

	// Pick the first workspace as the root.
	var workspace *insomniaResource
	for i := range doc.Resources {
		if doc.Resources[i].Type == "workspace" {
			workspace = &doc.Resources[i]
			break
		}
	}
	out := &entity.ImportedCollection{FormatLabel: "insomnia_v4"}
	if workspace != nil {
		out.Name = strings.TrimSpace(workspace.Name)
		out.Description = strings.TrimSpace(workspace.Description)
	}
	if out.Name == "" {
		out.Name = "Imported collection"
	}

	// First base environment: in Insomnia, base env is the root with subEnvironmentParentId
	// pointing to a workspace. We just grab the first environment resource for simplicity.
	for i := range doc.Resources {
		if doc.Resources[i].Type == "environment" {
			for k, v := range doc.Resources[i].Data {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				out.Variables = append(out.Variables, entity.ImportedVariable{
					Key: key, Value: insomniaAnyToString(v),
				})
			}
			// Stable variable order helps deterministic tests + UI.
			sort.Slice(out.Variables, func(i, j int) bool { return out.Variables[i].Key < out.Variables[j].Key })
			break
		}
	}

	warn := &warningSink{}

	// Build adjacency: parentID → ordered children (request_group + request), by metaSortKey.
	// Insomnia v4 has "metaSortKey" on request/group entries for stable ordering.
	children := make(map[string][]*insomniaResource)
	for i := range doc.Resources {
		r := &doc.Resources[i]
		if r.Type != "request" && r.Type != "request_group" {
			continue
		}
		children[r.ParentID] = append(children[r.ParentID], r)
	}
	for pid, list := range children {
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].MetaSortKey < list[j].MetaSortKey
		})
		children[pid] = list
	}

	var rootParentID string
	if workspace != nil {
		rootParentID = workspace.ID
	}
	for _, c := range children[rootParentID] {
		if item, ok := convertInsomniaNode(c, children, warn); ok {
			out.RootItems = append(out.RootItems, item)
		}
	}
	// Fallback: requests/groups not linked to any discovered workspace still shouldn't be lost.
	if len(out.RootItems) == 0 {
		for _, list := range children {
			for _, c := range list {
				if item, ok := convertInsomniaNode(c, children, warn); ok {
					out.RootItems = append(out.RootItems, item)
				}
			}
		}
	}

	out.Warnings = warn.list
	if len(out.RootItems) == 0 {
		return nil, errors.New("insomnia export has no importable items")
	}
	return out, nil
}

type insomniaResource struct {
	ID             string                      `json:"_id"`
	Type           string                      `json:"_type"`
	Name           string                      `json:"name,omitempty"`
	Description    string                      `json:"description,omitempty"`
	ParentID       string                      `json:"parentId,omitempty"`
	MetaSortKey    float64                     `json:"metaSortKey,omitempty"`
	Method         string                      `json:"method,omitempty"`
	URL            string                      `json:"url,omitempty"`
	Parameters     []insomniaKV                `json:"parameters,omitempty"`
	Headers        []insomniaKV                `json:"headers,omitempty"`
	Body           *insomniaBody               `json:"body,omitempty"`
	Authentication *insomniaAuth               `json:"authentication,omitempty"`
	Data           map[string]interface{}      `json:"data,omitempty"` // environment
}

type insomniaKV struct {
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
	Type     string `json:"type,omitempty"` // "file" for multipart
	FileName string `json:"fileName,omitempty"`
}

type insomniaBody struct {
	MimeType string       `json:"mimeType,omitempty"`
	Text     string       `json:"text,omitempty"`
	Params   []insomniaKV `json:"params,omitempty"`
}

type insomniaAuth struct {
	Type     string `json:"type,omitempty"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
	AddTo    string `json:"addTo,omitempty"` // "header" | "queryParams"
	Disabled bool   `json:"disabled,omitempty"`
}

func convertInsomniaNode(
	r *insomniaResource,
	children map[string][]*insomniaResource,
	warn *warningSink,
) (entity.ImportedItem, bool) {
	switch r.Type {
	case "request_group":
		folder := &entity.ImportedFolder{
			Name:        strings.TrimSpace(r.Name),
			Description: strings.TrimSpace(r.Description),
		}
		if folder.Name == "" {
			folder.Name = "Folder"
		}
		for _, c := range children[r.ID] {
			if item, ok := convertInsomniaNode(c, children, warn); ok {
				folder.Items = append(folder.Items, item)
			}
		}
		return entity.ImportedItem{Folder: folder}, true
	case "request":
		req := convertInsomniaRequest(r, warn)
		if req == nil {
			return entity.ImportedItem{}, false
		}
		return entity.ImportedItem{Request: req}, true
	}
	return entity.ImportedItem{}, false
}

func convertInsomniaRequest(r *insomniaResource, warn *warningSink) *entity.ImportedRequest {
	out := &entity.ImportedRequest{
		Name:   strings.TrimSpace(r.Name),
		Method: strings.ToUpper(strings.TrimSpace(r.Method)),
		URL:    strings.TrimSpace(r.URL),
	}
	if out.Name == "" {
		out.Name = "Request"
	}
	if out.Method == "" {
		out.Method = "GET"
	}

	for _, h := range r.Headers {
		if h.Disabled {
			continue
		}
		key := strings.TrimSpace(h.Name)
		if key == "" {
			continue
		}
		out.Headers = append(out.Headers, entity.KeyValue{Key: key, Value: h.Value})
	}
	for _, q := range r.Parameters {
		if q.Disabled {
			continue
		}
		key := strings.TrimSpace(q.Name)
		if key == "" {
			continue
		}
		out.QueryParams = append(out.QueryParams, entity.KeyValue{Key: key, Value: q.Value})
	}

	if r.Body != nil {
		mime := strings.ToLower(strings.TrimSpace(r.Body.MimeType))
		switch {
		case mime == "application/json" || strings.HasSuffix(mime, "+json"):
			s := r.Body.Text
			out.RawBody = &s
			out.BodyMode = string(entity.BodyModeRaw)
		case mime == "application/xml" || strings.Contains(mime, "xml"):
			s := r.Body.Text
			out.RawBody = &s
			out.BodyMode = string(entity.BodyModeXML)
		case mime == "application/x-www-form-urlencoded":
			out.BodyMode = string(entity.BodyModeFormURLEncoded)
			for _, p := range r.Body.Params {
				if p.Disabled {
					continue
				}
				key := strings.TrimSpace(p.Name)
				if key == "" {
					continue
				}
				out.FormFields = append(out.FormFields, entity.KeyValue{Key: key, Value: p.Value})
			}
		case mime == "multipart/form-data":
			out.BodyMode = string(entity.BodyModeMultipartFormData)
			for _, p := range r.Body.Params {
				if p.Disabled {
					continue
				}
				key := strings.TrimSpace(p.Name)
				if key == "" {
					continue
				}
				if strings.EqualFold(p.Type, "file") {
					warn.add(fmt.Sprintf("%s: multipart file %q skipped", out.Name, key))
					continue
				}
				out.MultipartParts = append(out.MultipartParts, entity.MultipartPart{
					Key: key, Kind: "text", Value: p.Value,
				})
			}
		case strings.TrimSpace(r.Body.Text) != "":
			s := r.Body.Text
			out.RawBody = &s
			out.BodyMode = string(entity.BodyModeRaw)
		default:
			out.BodyMode = string(entity.BodyModeNone)
		}
	} else {
		out.BodyMode = string(entity.BodyModeNone)
	}

	if r.Authentication != nil && !r.Authentication.Disabled {
		out.Auth = insomniaAuthToRequestAuth(r.Authentication)
	}
	return out
}

func insomniaAuthToRequestAuth(a *insomniaAuth) *entity.RequestAuth {
	switch strings.ToLower(strings.TrimSpace(a.Type)) {
	case "bearer":
		return &entity.RequestAuth{Type: "bearer", BearerToken: a.Token}
	case "basic":
		return &entity.RequestAuth{Type: "basic", Username: a.Username, Password: a.Password}
	case "apikey":
		in := strings.ToLower(strings.TrimSpace(a.AddTo))
		if in == "queryparams" || in == "query" {
			in = "query"
		} else {
			in = "header"
		}
		return &entity.RequestAuth{
			Type: "apikey", APIKeyName: a.Key, APIKey: a.Value, APIKeyIn: in,
		}
	case "none", "":
		return nil
	}
	return nil
}

func insomniaAnyToString(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case float64:
		// Trim trailing zeros for integer-looking values.
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", val), "0"), ".")
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
