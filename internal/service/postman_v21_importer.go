package service

import (
	"PostmanJanai/internal/entity"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// importPostmanV21 parses a Postman Collection v2.1 JSON document into an ImportedCollection.
// Reference: https://schema.postman.com/json/collection/v2.1.0/collection.json
func importPostmanV21(raw []byte) (*entity.ImportedCollection, error) {
	var doc postmanV21Collection
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse postman v2.1 json: %w", err)
	}
	if doc.Info.Name == "" && len(doc.Item) == 0 {
		return nil, errors.New("document does not look like a postman v2.1 collection")
	}

	out := &entity.ImportedCollection{
		Name:        strings.TrimSpace(doc.Info.Name),
		Description: strings.TrimSpace(stringOrEmpty(doc.Info.Description)),
		FormatLabel: "postman_v2.1",
	}
	if out.Name == "" {
		out.Name = "Imported collection"
	}

	for _, v := range doc.Variable {
		key := strings.TrimSpace(v.Key)
		if key == "" {
			continue
		}
		out.Variables = append(out.Variables, entity.ImportedVariable{Key: key, Value: stringOrEmpty(v.Value)})
	}

	// Inherit collection-level auth for requests that do not override it.
	parentAuth := parsePostmanAuth(doc.Auth)

	warn := &warningSink{}
	for _, it := range doc.Item {
		child, ok := convertPostmanV21Item(it, parentAuth, warn)
		if ok {
			out.RootItems = append(out.RootItems, child)
		}
	}
	out.Warnings = warn.list

	if len(out.RootItems) == 0 {
		return nil, errors.New("collection has no importable items")
	}
	return out, nil
}

type postmanV21Collection struct {
	Info     postmanV21Info     `json:"info"`
	Item     []postmanV21Item   `json:"item"`
	Auth     *postmanAuth       `json:"auth,omitempty"`
	Variable []postmanV21KV     `json:"variable,omitempty"`
}

type postmanV21Info struct {
	Name        string          `json:"name"`
	Description json.RawMessage `json:"description,omitempty"`
	Schema      string          `json:"schema,omitempty"`
}

type postmanV21Item struct {
	Name        string          `json:"name,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
	Item        []postmanV21Item `json:"item,omitempty"`
	Request     *postmanV21Req  `json:"request,omitempty"`
	Auth        *postmanAuth    `json:"auth,omitempty"`
}

type postmanV21Req struct {
	Method      string          `json:"method,omitempty"`
	URL         json.RawMessage `json:"url,omitempty"`
	Header      []postmanV21KV  `json:"header,omitempty"`
	Body        *postmanV21Body `json:"body,omitempty"`
	Auth        *postmanAuth    `json:"auth,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
}

type postmanV21KV struct {
	Key      string          `json:"key,omitempty"`
	Value    json.RawMessage `json:"value,omitempty"`
	Type     string          `json:"type,omitempty"`
	Src      json.RawMessage `json:"src,omitempty"`
	Disabled bool            `json:"disabled,omitempty"`
}

type postmanV21Body struct {
	Mode       string           `json:"mode,omitempty"`
	Raw        string           `json:"raw,omitempty"`
	Options    *postmanV21BodyOpts `json:"options,omitempty"`
	URLEncoded []postmanV21KV   `json:"urlencoded,omitempty"`
	FormData   []postmanV21KV   `json:"formdata,omitempty"`
	File       json.RawMessage  `json:"file,omitempty"`
	GraphQL    json.RawMessage  `json:"graphql,omitempty"`
}

type postmanV21BodyOpts struct {
	Raw *postmanV21RawOpt `json:"raw,omitempty"`
}

type postmanV21RawOpt struct {
	Language string `json:"language,omitempty"`
}

type postmanAuth struct {
	Type   string          `json:"type,omitempty"`
	Bearer json.RawMessage `json:"bearer,omitempty"`
	Basic  json.RawMessage `json:"basic,omitempty"`
	APIKey json.RawMessage `json:"apikey,omitempty"`
}

type postmanAuthPair struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	Type  string          `json:"type,omitempty"`
}

func convertPostmanV21Item(it postmanV21Item, parentAuth *entity.RequestAuth, warn *warningSink) (entity.ImportedItem, bool) {
	if it.Request != nil {
		req := convertPostmanV21Request(it, parentAuth, warn)
		if req == nil {
			return entity.ImportedItem{}, false
		}
		return entity.ImportedItem{Request: req}, true
	}
	// Folder when it has nested items (even empty folder allowed).
	folder := &entity.ImportedFolder{
		Name:        strings.TrimSpace(it.Name),
		Description: strings.TrimSpace(stringOrEmpty(it.Description)),
	}
	if folder.Name == "" {
		folder.Name = "Folder"
	}
	// Folder-level auth cascades down to children when request auth is missing.
	childAuth := parentAuth
	if it.Auth != nil {
		if a := parsePostmanAuth(it.Auth); a != nil {
			childAuth = a
		}
	}
	for _, child := range it.Item {
		c, ok := convertPostmanV21Item(child, childAuth, warn)
		if ok {
			folder.Items = append(folder.Items, c)
		}
	}
	return entity.ImportedItem{Folder: folder}, true
}

func convertPostmanV21Request(it postmanV21Item, parentAuth *entity.RequestAuth, warn *warningSink) *entity.ImportedRequest {
	r := it.Request
	out := &entity.ImportedRequest{
		Name:   strings.TrimSpace(it.Name),
		Method: strings.ToUpper(strings.TrimSpace(r.Method)),
	}
	if out.Name == "" {
		out.Name = "Request"
	}
	if out.Method == "" {
		out.Method = "GET"
	}

	urlStr, query, pathVars := parsePostmanURL(r.URL)
	out.URL = urlStr
	out.QueryParams = query
	if len(pathVars) > 0 {
		warn.add(fmt.Sprintf("%s: path variables preserved as-is in URL", out.Name))
	}

	for _, h := range r.Header {
		if h.Disabled {
			continue
		}
		key := strings.TrimSpace(h.Key)
		if key == "" {
			continue
		}
		out.Headers = append(out.Headers, entity.KeyValue{Key: key, Value: stringOrEmpty(h.Value)})
	}

	auth := parsePostmanAuth(r.Auth)
	if auth == nil {
		auth = parentAuth
	}
	out.Auth = auth

	if r.Body != nil {
		applyPostmanBody(out, r.Body, warn)
	} else {
		out.BodyMode = string(entity.BodyModeNone)
	}
	return out
}

func applyPostmanBody(out *entity.ImportedRequest, body *postmanV21Body, warn *warningSink) {
	switch strings.ToLower(strings.TrimSpace(body.Mode)) {
	case "raw":
		raw := body.Raw
		lang := ""
		if body.Options != nil && body.Options.Raw != nil {
			lang = strings.ToLower(strings.TrimSpace(body.Options.Raw.Language))
		}
		if lang == "xml" || strings.HasPrefix(strings.TrimSpace(raw), "<?xml") {
			out.BodyMode = string(entity.BodyModeXML)
		} else {
			out.BodyMode = string(entity.BodyModeRaw)
		}
		s := raw
		out.RawBody = &s
	case "urlencoded":
		out.BodyMode = string(entity.BodyModeFormURLEncoded)
		for _, kv := range body.URLEncoded {
			if kv.Disabled {
				continue
			}
			k := strings.TrimSpace(kv.Key)
			if k == "" {
				continue
			}
			out.FormFields = append(out.FormFields, entity.KeyValue{Key: k, Value: stringOrEmpty(kv.Value)})
		}
	case "formdata":
		out.BodyMode = string(entity.BodyModeMultipartFormData)
		for _, kv := range body.FormData {
			if kv.Disabled {
				continue
			}
			k := strings.TrimSpace(kv.Key)
			if k == "" {
				continue
			}
			kind := strings.ToLower(strings.TrimSpace(kv.Type))
			if kind == "file" {
				warn.add(fmt.Sprintf("%s: multipart file %q skipped (file path not portable)", out.Name, k))
				continue
			}
			out.MultipartParts = append(out.MultipartParts, entity.MultipartPart{
				Key: k, Kind: "text", Value: stringOrEmpty(kv.Value),
			})
		}
	case "file", "binary":
		warn.add(fmt.Sprintf("%s: %q body mode not supported, skipped", out.Name, body.Mode))
		out.BodyMode = string(entity.BodyModeNone)
	case "graphql":
		warn.add(fmt.Sprintf("%s: graphql body imported as raw JSON", out.Name))
		if len(body.GraphQL) > 0 {
			s := string(body.GraphQL)
			out.RawBody = &s
			out.BodyMode = string(entity.BodyModeRaw)
		} else {
			out.BodyMode = string(entity.BodyModeNone)
		}
	default:
		out.BodyMode = string(entity.BodyModeNone)
	}
}

// parsePostmanURL handles both the string shortcut and the structured URL object.
// Returns the `raw` URL (preserving {{var}}), plus enabled query params and path variables.
func parsePostmanURL(raw json.RawMessage) (string, []entity.KeyValue, []string) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	// String shortcut: "https://host/path?x=1"
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil && asString != "" {
		return asString, parseQueryParamsFromURLString(asString), nil
	}
	var obj struct {
		Raw      string          `json:"raw,omitempty"`
		Protocol string          `json:"protocol,omitempty"`
		Host     json.RawMessage `json:"host,omitempty"`
		Path     json.RawMessage `json:"path,omitempty"`
		Port     string          `json:"port,omitempty"`
		Query    []postmanV21KV  `json:"query,omitempty"`
		Variable []postmanV21KV  `json:"variable,omitempty"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", nil, nil
	}
	var query []entity.KeyValue
	for _, q := range obj.Query {
		if q.Disabled {
			continue
		}
		k := strings.TrimSpace(q.Key)
		if k == "" {
			continue
		}
		query = append(query, entity.KeyValue{Key: k, Value: stringOrEmpty(q.Value)})
	}
	var pathVars []string
	for _, v := range obj.Variable {
		k := strings.TrimSpace(v.Key)
		if k != "" {
			pathVars = append(pathVars, k)
		}
	}
	if strings.TrimSpace(obj.Raw) != "" {
		return obj.Raw, query, pathVars
	}
	// Rebuild from parts when raw missing.
	rebuilt := buildURLFromParts(obj.Protocol, obj.Host, obj.Path, obj.Port)
	return rebuilt, query, pathVars
}

func buildURLFromParts(protocol string, host, path json.RawMessage, port string) string {
	protocol = strings.TrimSpace(protocol)
	if protocol == "" {
		protocol = "https"
	}
	hostStr := joinStringOrArray(host, ".")
	pathStr := joinStringOrArray(path, "/")
	if hostStr == "" && pathStr == "" {
		return ""
	}
	sb := strings.Builder{}
	sb.WriteString(protocol)
	sb.WriteString("://")
	sb.WriteString(hostStr)
	if strings.TrimSpace(port) != "" {
		sb.WriteString(":")
		sb.WriteString(strings.TrimSpace(port))
	}
	if pathStr != "" {
		if !strings.HasPrefix(pathStr, "/") {
			sb.WriteString("/")
		}
		sb.WriteString(pathStr)
	}
	return sb.String()
}

// joinStringOrArray accepts either "a.b.c" or ["a","b","c"] and joins with sep.
func joinStringOrArray(raw json.RawMessage, sep string) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		parts := make([]string, 0, len(arr))
		for _, p := range arr {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, p)
			}
		}
		return strings.Join(parts, sep)
	}
	return ""
}

// parseQueryParamsFromURLString pulls query key/value pairs from "https://x?a=1&b=2".
// Keeps Postman {{var}} markers literal (url.Parse handles them fine inside query strings).
func parseQueryParamsFromURLString(s string) []entity.KeyValue {
	idx := strings.Index(s, "?")
	if idx < 0 || idx == len(s)-1 {
		return nil
	}
	q := s[idx+1:]
	vals, err := url.ParseQuery(q)
	if err != nil {
		return nil
	}
	var out []entity.KeyValue
	for k, arr := range vals {
		for _, v := range arr {
			out = append(out, entity.KeyValue{Key: k, Value: v})
		}
	}
	return out
}

func parsePostmanAuth(pa *postmanAuth) *entity.RequestAuth {
	if pa == nil {
		return nil
	}
	kind := strings.ToLower(strings.TrimSpace(pa.Type))
	switch kind {
	case "":
		return nil
	case "noauth":
		return &entity.RequestAuth{Type: "none"}
	case "bearer":
		token := postmanAuthField(pa.Bearer, "token")
		return &entity.RequestAuth{Type: "bearer", BearerToken: token}
	case "basic":
		user := postmanAuthField(pa.Basic, "username")
		pass := postmanAuthField(pa.Basic, "password")
		return &entity.RequestAuth{Type: "basic", Username: user, Password: pass}
	case "apikey":
		key := postmanAuthField(pa.APIKey, "key")
		value := postmanAuthField(pa.APIKey, "value")
		in := strings.ToLower(postmanAuthField(pa.APIKey, "in"))
		if in != "query" {
			in = "header"
		}
		return &entity.RequestAuth{Type: "apikey", APIKeyName: key, APIKey: value, APIKeyIn: in}
	default:
		return nil
	}
}

// postmanAuthField extracts the {"key":"X","value":"Y"} pair from Postman's auth list/map.
// Accepts both array-of-pairs (v2.1) and object map (fallback) shapes.
func postmanAuthField(raw json.RawMessage, want string) string {
	if len(raw) == 0 {
		return ""
	}
	var pairs []postmanAuthPair
	if err := json.Unmarshal(raw, &pairs); err == nil {
		for _, p := range pairs {
			if strings.EqualFold(p.Key, want) {
				return stringOrEmpty(p.Value)
			}
		}
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		for k, v := range obj {
			if strings.EqualFold(k, want) {
				return stringOrEmpty(v)
			}
		}
	}
	return ""
}

// stringOrEmpty unmarshals a raw JSON value as string; empty if null/non-string/missing.
func stringOrEmpty(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Accept numbers / bools by falling back to the literal representation.
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" {
		return ""
	}
	return trimmed
}

// warningSink deduplicates non-fatal warnings while preserving insertion order.
type warningSink struct {
	list []string
	seen map[string]struct{}
}

func (w *warningSink) add(s string) {
	if w.seen == nil {
		w.seen = make(map[string]struct{})
	}
	if _, ok := w.seen[s]; ok {
		return
	}
	w.seen[s] = struct{}{}
	w.list = append(w.list, s)
}
