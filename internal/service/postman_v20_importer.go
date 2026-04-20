package service

import (
	"PostmanJanai/internal/entity"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// importPostmanV20 parses Postman Collection v2.0 (legacy) JSON. v2.0 stores a flat `folders` and
// `requests` list plus `order` / `folders_order` arrays to reconstruct the tree. This importer
// aims for a pragmatic, lossy mapping suitable for the app (best-effort; raw body + form data).
func importPostmanV20(raw []byte) (*entity.ImportedCollection, error) {
	var doc postmanV20Collection
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse postman v2.0 json: %w", err)
	}
	if doc.Name == "" && len(doc.Requests) == 0 && len(doc.Folders) == 0 {
		return nil, errors.New("document does not look like a postman v2.0 collection")
	}

	out := &entity.ImportedCollection{
		Name:        strings.TrimSpace(doc.Name),
		Description: strings.TrimSpace(doc.Description),
		FormatLabel: "postman_v2.0",
	}
	if out.Name == "" {
		out.Name = "Imported collection"
	}

	for _, v := range doc.Variables {
		key := strings.TrimSpace(v.Key)
		if key == "" {
			continue
		}
		out.Variables = append(out.Variables, entity.ImportedVariable{Key: key, Value: v.Value})
	}

	warn := &warningSink{}

	// Build lookup maps.
	folderByID := make(map[string]*postmanV20Folder, len(doc.Folders))
	for i := range doc.Folders {
		folderByID[doc.Folders[i].ID] = &doc.Folders[i]
	}
	reqByID := make(map[string]*postmanV20Request, len(doc.Requests))
	for i := range doc.Requests {
		reqByID[doc.Requests[i].ID] = &doc.Requests[i]
	}

	// Assemble tree: start from collection.order (request IDs at root) + folders at root (those
	// with no parent folder). Postman v2.0 puts folders under collection, referenced by name in
	// the `requests[].folder` field. We rely primarily on `folder` on each request, and on
	// `folders_order`/`order` inside folders for nesting among folders.
	rootFolderIDs := findRootFolderIDs(doc.Folders)
	for _, fid := range rootFolderIDs {
		f := folderByID[fid]
		if f == nil {
			continue
		}
		item, ok := convertPostmanV20Folder(f, folderByID, reqByID, warn)
		if ok {
			out.RootItems = append(out.RootItems, item)
		}
	}
	// Requests at root (no folder).
	for _, rid := range doc.Order {
		r := reqByID[rid]
		if r == nil || strings.TrimSpace(r.Folder) != "" {
			continue
		}
		req := convertPostmanV20Request(r, warn)
		if req != nil {
			out.RootItems = append(out.RootItems, entity.ImportedItem{Request: req})
		}
	}
	// Fallback: include any request that wasn't referenced by order + has no folder.
	seenInOrder := make(map[string]struct{}, len(doc.Order))
	for _, rid := range doc.Order {
		seenInOrder[rid] = struct{}{}
	}
	for _, r := range doc.Requests {
		if strings.TrimSpace(r.Folder) != "" {
			continue
		}
		if _, ok := seenInOrder[r.ID]; ok {
			continue
		}
		req := convertPostmanV20Request(&r, warn)
		if req != nil {
			out.RootItems = append(out.RootItems, entity.ImportedItem{Request: req})
		}
	}

	out.Warnings = warn.list
	if len(out.RootItems) == 0 {
		return nil, errors.New("collection has no importable items")
	}
	return out, nil
}

type postmanV20Collection struct {
	ID          string                `json:"id,omitempty"`
	Name        string                `json:"name,omitempty"`
	Description string                `json:"description,omitempty"`
	Order       []string              `json:"order,omitempty"`
	Folders     []postmanV20Folder    `json:"folders,omitempty"`
	Requests    []postmanV20Request   `json:"requests,omitempty"`
	Variables   []postmanV20Variable  `json:"variables,omitempty"`
}

type postmanV20Variable struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type postmanV20Folder struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name,omitempty"`
	Description   string   `json:"description,omitempty"`
	Order         []string `json:"order,omitempty"`         // request IDs in this folder, ordered
	FoldersOrder  []string `json:"folders_order,omitempty"` // sub-folder IDs, ordered
	ParentFolder  string   `json:"folder,omitempty"`        // parent folder id when nested
}

type postmanV20Request struct {
	ID           string                   `json:"id,omitempty"`
	Name         string                   `json:"name,omitempty"`
	Method       string                   `json:"method,omitempty"`
	URL          string                   `json:"url,omitempty"`
	Headers      string                   `json:"headers,omitempty"`       // legacy: newline separated
	HeaderData   []postmanV20KV           `json:"headerData,omitempty"`    // newer
	QueryParams  []postmanV20KV           `json:"queryParams,omitempty"`
	DataMode     string                   `json:"dataMode,omitempty"`      // raw | params | urlencoded | binary
	Data         []postmanV20KV           `json:"data,omitempty"`          // urlencoded / form-data
	RawModeData  string                   `json:"rawModeData,omitempty"`
	Folder       string                   `json:"folder,omitempty"`
}

type postmanV20KV struct {
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
	Type     string `json:"type,omitempty"` // text | file
	Enabled  *bool  `json:"enabled,omitempty"`
}

// findRootFolderIDs returns folder IDs that have no parent folder reference.
func findRootFolderIDs(folders []postmanV20Folder) []string {
	var out []string
	for _, f := range folders {
		if strings.TrimSpace(f.ParentFolder) == "" {
			out = append(out, f.ID)
		}
	}
	return out
}

func convertPostmanV20Folder(
	f *postmanV20Folder,
	folderByID map[string]*postmanV20Folder,
	reqByID map[string]*postmanV20Request,
	warn *warningSink,
) (entity.ImportedItem, bool) {
	folder := &entity.ImportedFolder{
		Name:        strings.TrimSpace(f.Name),
		Description: strings.TrimSpace(f.Description),
	}
	if folder.Name == "" {
		folder.Name = "Folder"
	}
	// Sub-folders first (ordered by folders_order if provided).
	for _, sid := range f.FoldersOrder {
		sub := folderByID[sid]
		if sub == nil {
			continue
		}
		if item, ok := convertPostmanV20Folder(sub, folderByID, reqByID, warn); ok {
			folder.Items = append(folder.Items, item)
		}
	}
	// Requests in folder (ordered by `order`).
	for _, rid := range f.Order {
		r := reqByID[rid]
		if r == nil {
			continue
		}
		if req := convertPostmanV20Request(r, warn); req != nil {
			folder.Items = append(folder.Items, entity.ImportedItem{Request: req})
		}
	}
	// Fallback: include any request referencing this folder but not listed in `order`.
	seen := make(map[string]struct{}, len(f.Order))
	for _, rid := range f.Order {
		seen[rid] = struct{}{}
	}
	for _, r := range reqByID {
		if r == nil || r.Folder != f.ID {
			continue
		}
		if _, ok := seen[r.ID]; ok {
			continue
		}
		if req := convertPostmanV20Request(r, warn); req != nil {
			folder.Items = append(folder.Items, entity.ImportedItem{Request: req})
		}
	}
	return entity.ImportedItem{Folder: folder}, true
}

func convertPostmanV20Request(r *postmanV20Request, warn *warningSink) *entity.ImportedRequest {
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

	// Headers: prefer structured headerData; fall back to parsing legacy `headers` blob.
	if len(r.HeaderData) > 0 {
		for _, h := range r.HeaderData {
			if h.Enabled != nil && !*h.Enabled {
				continue
			}
			k := strings.TrimSpace(h.Key)
			if k == "" {
				continue
			}
			out.Headers = append(out.Headers, entity.KeyValue{Key: k, Value: h.Value})
		}
	} else if strings.TrimSpace(r.Headers) != "" {
		for _, line := range strings.Split(r.Headers, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			idx := strings.Index(line, ":")
			if idx <= 0 {
				continue
			}
			out.Headers = append(out.Headers, entity.KeyValue{
				Key:   strings.TrimSpace(line[:idx]),
				Value: strings.TrimSpace(line[idx+1:]),
			})
		}
	}

	for _, q := range r.QueryParams {
		if q.Enabled != nil && !*q.Enabled {
			continue
		}
		k := strings.TrimSpace(q.Key)
		if k == "" {
			continue
		}
		out.QueryParams = append(out.QueryParams, entity.KeyValue{Key: k, Value: q.Value})
	}

	switch strings.ToLower(strings.TrimSpace(r.DataMode)) {
	case "raw":
		body := r.RawModeData
		if strings.HasPrefix(strings.TrimSpace(body), "<?xml") {
			out.BodyMode = string(entity.BodyModeXML)
		} else {
			out.BodyMode = string(entity.BodyModeRaw)
		}
		out.RawBody = &body
	case "urlencoded":
		out.BodyMode = string(entity.BodyModeFormURLEncoded)
		for _, kv := range r.Data {
			if kv.Enabled != nil && !*kv.Enabled {
				continue
			}
			k := strings.TrimSpace(kv.Key)
			if k == "" {
				continue
			}
			out.FormFields = append(out.FormFields, entity.KeyValue{Key: k, Value: kv.Value})
		}
	case "params":
		out.BodyMode = string(entity.BodyModeMultipartFormData)
		for _, kv := range r.Data {
			if kv.Enabled != nil && !*kv.Enabled {
				continue
			}
			k := strings.TrimSpace(kv.Key)
			if k == "" {
				continue
			}
			if strings.EqualFold(kv.Type, "file") {
				warn.add(fmt.Sprintf("%s: multipart file %q skipped", out.Name, k))
				continue
			}
			out.MultipartParts = append(out.MultipartParts, entity.MultipartPart{
				Key: k, Kind: "text", Value: kv.Value,
			})
		}
	case "binary":
		warn.add(fmt.Sprintf("%s: binary body not supported, skipped", out.Name))
		out.BodyMode = string(entity.BodyModeNone)
	default:
		out.BodyMode = string(entity.BodyModeNone)
	}
	return out
}
