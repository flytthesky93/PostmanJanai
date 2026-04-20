package service

import (
	"PostmanJanai/internal/entity"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// importOpenAPI parses an OpenAPI 3.x document (JSON or YAML) into an ImportedCollection.
// Mapping strategy:
//   - info.title  → root folder name; info.description → root description
//   - tags        → sub-folders; operations without tag → placed under root
//   - paths/{p}/{method} → request with template URL "{{baseUrl}}{p}"
//   - parameters (query/header) → Headers / QueryParams (path params kept literal in URL)
//   - requestBody application/json → raw JSON from example (first available); other media → warn
//   - security (bearer / basic / apiKey) → RequestAuth
//   - servers[0].url → exposed as a collection variable "baseUrl" (seed into env when user opts in)
//
// This is intentionally pragmatic: an exhaustive OpenAPI-to-request mapping is out of scope.
func importOpenAPI(raw []byte) (*entity.ImportedCollection, error) {
	doc, err := decodeOpenAPIDoc(raw)
	if err != nil {
		return nil, err
	}
	if !looksLikeOpenAPI(doc) {
		return nil, errors.New("document is not a recognized OpenAPI 3 file")
	}

	out := &entity.ImportedCollection{
		Name:        strings.TrimSpace(doc.Info.Title),
		Description: strings.TrimSpace(doc.Info.Description),
		FormatLabel: "openapi_3.x",
	}
	if out.Name == "" {
		out.Name = "OpenAPI collection"
	}

	// Default base URL variable so UI requests use "{{baseUrl}}/..." after import.
	baseURL := ""
	if len(doc.Servers) > 0 {
		baseURL = strings.TrimRight(strings.TrimSpace(doc.Servers[0].URL), "/")
	}
	if baseURL == "" {
		baseURL = "https://"
	}
	out.Variables = append(out.Variables, entity.ImportedVariable{Key: "baseUrl", Value: baseURL})

	// Resolve shared security schemes (bearer / basic / apikey only).
	security := parseSecuritySchemes(doc)
	defaultAuth := resolveSecurityRequirement(doc.Security, security)

	warn := &warningSink{}

	// Group operations by tag (first tag wins); operations without tag go under root directly.
	folderMap := make(map[string]*entity.ImportedFolder)
	var rootRequests []entity.ImportedItem

	// Stable path order for deterministic output.
	pathKeys := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	for _, pathTmpl := range pathKeys {
		item := doc.Paths[pathTmpl]
		// Path-level parameters apply to every operation on this path.
		pathParams := item.Parameters
		for _, method := range openAPIMethodOrder() {
			op := item.Operation(method)
			if op == nil {
				continue
			}
			req := convertOpenAPIOperation(pathTmpl, method, op, pathParams, security, defaultAuth, warn)
			if req == nil {
				continue
			}
			tag := ""
			if len(op.Tags) > 0 {
				tag = strings.TrimSpace(op.Tags[0])
			}
			if tag == "" {
				rootRequests = append(rootRequests, entity.ImportedItem{Request: req})
				continue
			}
			folder, ok := folderMap[tag]
			if !ok {
				folder = &entity.ImportedFolder{Name: tag}
				folderMap[tag] = folder
			}
			folder.Items = append(folder.Items, entity.ImportedItem{Request: req})
		}
	}

	// Stable folder order: by tag order in doc.Tags first, then alphabetical for the rest.
	seen := make(map[string]struct{}, len(folderMap))
	for _, t := range doc.Tags {
		name := strings.TrimSpace(t.Name)
		if f, ok := folderMap[name]; ok {
			if t.Description != "" {
				f.Description = t.Description
			}
			out.RootItems = append(out.RootItems, entity.ImportedItem{Folder: f})
			seen[name] = struct{}{}
		}
	}
	leftover := make([]string, 0)
	for name := range folderMap {
		if _, ok := seen[name]; !ok {
			leftover = append(leftover, name)
		}
	}
	sort.Strings(leftover)
	for _, name := range leftover {
		out.RootItems = append(out.RootItems, entity.ImportedItem{Folder: folderMap[name]})
	}
	out.RootItems = append(out.RootItems, rootRequests...)

	out.Warnings = warn.list
	if len(out.RootItems) == 0 {
		return nil, errors.New("OpenAPI document has no paths")
	}
	return out, nil
}

// --- OpenAPI types (minimal subset we care about) ---

type openAPIDoc struct {
	OpenAPI    string                     `json:"openapi,omitempty" yaml:"openapi,omitempty"`
	Swagger    string                     `json:"swagger,omitempty" yaml:"swagger,omitempty"`
	Info       openAPIInfo                `json:"info" yaml:"info"`
	Servers    []openAPIServer            `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []openAPITag               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Paths      map[string]openAPIPathItem `json:"paths" yaml:"paths"`
	Components *openAPIComponents         `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []map[string][]string      `json:"security,omitempty" yaml:"security,omitempty"`
}

type openAPIInfo struct {
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
}

type openAPIServer struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type openAPITag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type openAPIPathItem struct {
	Parameters []openAPIParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Get        *openAPIOperation  `json:"get,omitempty" yaml:"get,omitempty"`
	Post       *openAPIOperation  `json:"post,omitempty" yaml:"post,omitempty"`
	Put        *openAPIOperation  `json:"put,omitempty" yaml:"put,omitempty"`
	Delete     *openAPIOperation  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch      *openAPIOperation  `json:"patch,omitempty" yaml:"patch,omitempty"`
	Options    *openAPIOperation  `json:"options,omitempty" yaml:"options,omitempty"`
	Head       *openAPIOperation  `json:"head,omitempty" yaml:"head,omitempty"`
	Trace      *openAPIOperation  `json:"trace,omitempty" yaml:"trace,omitempty"`
}

func (p openAPIPathItem) Operation(method string) *openAPIOperation {
	switch strings.ToUpper(method) {
	case "GET":
		return p.Get
	case "POST":
		return p.Post
	case "PUT":
		return p.Put
	case "DELETE":
		return p.Delete
	case "PATCH":
		return p.Patch
	case "OPTIONS":
		return p.Options
	case "HEAD":
		return p.Head
	case "TRACE":
		return p.Trace
	}
	return nil
}

type openAPIOperation struct {
	Summary     string                 `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                 `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []openAPIParameter     `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *openAPIRequestBody    `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Security    []map[string][]string  `json:"security,omitempty" yaml:"security,omitempty"`
}

type openAPIParameter struct {
	Name     string          `json:"name" yaml:"name"`
	In       string          `json:"in" yaml:"in"`
	Required bool            `json:"required,omitempty" yaml:"required,omitempty"`
	Example  json.RawMessage `json:"example,omitempty" yaml:"example,omitempty"`
	Schema   json.RawMessage `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type openAPIRequestBody struct {
	Required bool                          `json:"required,omitempty" yaml:"required,omitempty"`
	Content  map[string]openAPIMediaType   `json:"content,omitempty" yaml:"content,omitempty"`
}

type openAPIMediaType struct {
	Schema   json.RawMessage            `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  json.RawMessage            `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]openAPIExample  `json:"examples,omitempty" yaml:"examples,omitempty"`
}

type openAPIExample struct {
	Summary string          `json:"summary,omitempty" yaml:"summary,omitempty"`
	Value   json.RawMessage `json:"value,omitempty" yaml:"value,omitempty"`
}

type openAPIComponents struct {
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type openAPISecurityScheme struct {
	Type         string `json:"type,omitempty" yaml:"type,omitempty"`          // http | apiKey | oauth2 | openIdConnect
	Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"`      // bearer | basic
	In           string `json:"in,omitempty" yaml:"in,omitempty"`              // header | query | cookie (apiKey)
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`          // apiKey header/query name
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
}

// --- helpers ---

func decodeOpenAPIDoc(raw []byte) (*openAPIDoc, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("empty openapi document")
	}
	jsonBytes := trimmed
	if trimmed[0] != '{' && trimmed[0] != '[' {
		// YAML input: decode to generic tree then re-serialize as JSON so that json.RawMessage
		// fields (examples, schemas) are populated correctly.
		var tree interface{}
		if err := yaml.Unmarshal(trimmed, &tree); err != nil {
			return nil, fmt.Errorf("parse openapi yaml: %w", err)
		}
		tree = normalizeYAMLToJSON(tree)
		b, err := json.Marshal(tree)
		if err != nil {
			return nil, fmt.Errorf("serialize openapi yaml: %w", err)
		}
		jsonBytes = b
	}
	var doc openAPIDoc
	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi json: %w", err)
	}
	return &doc, nil
}

// normalizeYAMLToJSON converts YAML-decoded map[interface{}]interface{} (common in some YAML
// libraries) to map[string]interface{} so encoding/json can marshal it. yaml.v3 returns
// map[string]interface{} by default, but we keep this robust for nested any-keyed maps too.
func normalizeYAMLToJSON(in interface{}) interface{} {
	switch v := in.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[fmt.Sprint(k)] = normalizeYAMLToJSON(val)
		}
		return out
	case map[string]interface{}:
		for k, val := range v {
			v[k] = normalizeYAMLToJSON(val)
		}
		return v
	case []interface{}:
		for i, val := range v {
			v[i] = normalizeYAMLToJSON(val)
		}
		return v
	default:
		return v
	}
}

func looksLikeOpenAPI(doc *openAPIDoc) bool {
	if doc == nil {
		return false
	}
	if strings.HasPrefix(doc.OpenAPI, "3.") {
		return true
	}
	// Accept swagger 2.0 filtering later if ever supported; for now require OpenAPI 3.x.
	return false
}

func openAPIMethodOrder() []string {
	return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"}
}

func parseSecuritySchemes(doc *openAPIDoc) map[string]openAPISecurityScheme {
	if doc.Components == nil {
		return nil
	}
	return doc.Components.SecuritySchemes
}

// resolveSecurityRequirement picks the first security requirement that maps to a supported scheme.
func resolveSecurityRequirement(
	reqs []map[string][]string,
	schemes map[string]openAPISecurityScheme,
) *entity.RequestAuth {
	for _, req := range reqs {
		for name := range req {
			if a := securitySchemeToAuth(schemes[name]); a != nil {
				return a
			}
		}
	}
	return nil
}

func securitySchemeToAuth(s openAPISecurityScheme) *entity.RequestAuth {
	switch strings.ToLower(strings.TrimSpace(s.Type)) {
	case "http":
		switch strings.ToLower(strings.TrimSpace(s.Scheme)) {
		case "bearer":
			return &entity.RequestAuth{Type: "bearer", BearerToken: "{{token}}"}
		case "basic":
			return &entity.RequestAuth{Type: "basic", Username: "{{username}}", Password: "{{password}}"}
		}
	case "apikey":
		in := strings.ToLower(strings.TrimSpace(s.In))
		if in != "query" {
			in = "header"
		}
		return &entity.RequestAuth{
			Type:       "apikey",
			APIKeyName: strings.TrimSpace(s.Name),
			APIKey:     "{{apiKey}}",
			APIKeyIn:   in,
		}
	}
	return nil
}

func convertOpenAPIOperation(
	pathTmpl, method string,
	op *openAPIOperation,
	pathLevelParams []openAPIParameter,
	schemes map[string]openAPISecurityScheme,
	defaultAuth *entity.RequestAuth,
	warn *warningSink,
) *entity.ImportedRequest {
	name := strings.TrimSpace(op.Summary)
	if name == "" {
		name = strings.TrimSpace(op.OperationID)
	}
	if name == "" {
		name = fmt.Sprintf("%s %s", strings.ToUpper(method), pathTmpl)
	}
	req := &entity.ImportedRequest{
		Name:   name,
		Method: strings.ToUpper(method),
		URL:    "{{baseUrl}}" + ensureLeadingSlash(pathTmpl),
	}

	// Merge path-level + operation-level parameters (path-level first for deterministic order).
	params := make([]openAPIParameter, 0, len(pathLevelParams)+len(op.Parameters))
	params = append(params, pathLevelParams...)
	params = append(params, op.Parameters...)

	for _, p := range params {
		switch strings.ToLower(p.In) {
		case "query":
			req.QueryParams = append(req.QueryParams, entity.KeyValue{
				Key:   p.Name,
				Value: openAPIExampleOrEmpty(p.Example, p.Schema),
			})
		case "header":
			req.Headers = append(req.Headers, entity.KeyValue{
				Key:   p.Name,
				Value: openAPIExampleOrEmpty(p.Example, p.Schema),
			})
		case "path":
			// Left literal as "{name}" in URL — user-visible placeholder, app won't resolve it
			// automatically. We intentionally don't rewrite to {{name}} because that would
			// collide with environment variable semantics.
			_ = p
		}
	}

	// Auth: operation-level overrides global; empty-security array means "no auth".
	if op.Security != nil {
		if len(op.Security) == 0 {
			req.Auth = &entity.RequestAuth{Type: "none"}
		} else {
			if a := resolveSecurityRequirement(op.Security, schemes); a != nil {
				req.Auth = a
			}
		}
	} else {
		req.Auth = defaultAuth
	}

	// Request body — prefer application/json example.
	if op.RequestBody != nil {
		if mt, ok := op.RequestBody.Content["application/json"]; ok {
			if body := openAPIExampleAsJSON(mt); body != "" {
				s := body
				req.RawBody = &s
				req.BodyMode = string(entity.BodyModeRaw)
				req.Headers = ensureContentTypeHeader(req.Headers, "application/json")
			} else {
				req.Headers = ensureContentTypeHeader(req.Headers, "application/json")
				req.BodyMode = string(entity.BodyModeNone)
				warn.add(fmt.Sprintf("%s: JSON body had no example; body left empty", req.Name))
			}
		} else if mt, ok := op.RequestBody.Content["application/x-www-form-urlencoded"]; ok {
			req.BodyMode = string(entity.BodyModeFormURLEncoded)
			for k, v := range openAPIFormFieldsFromSchema(mt.Schema) {
				req.FormFields = append(req.FormFields, entity.KeyValue{Key: k, Value: v})
			}
			sort.Slice(req.FormFields, func(i, j int) bool { return req.FormFields[i].Key < req.FormFields[j].Key })
		} else if _, ok := op.RequestBody.Content["multipart/form-data"]; ok {
			req.BodyMode = string(entity.BodyModeMultipartFormData)
			warn.add(fmt.Sprintf("%s: multipart body imported empty (add fields manually)", req.Name))
		} else {
			// Some other media type (xml, text, binary) — leave body empty and warn.
			for mediaType := range op.RequestBody.Content {
				warn.add(fmt.Sprintf("%s: body media %q not converted", req.Name, mediaType))
				break
			}
			req.BodyMode = string(entity.BodyModeNone)
		}
	} else {
		req.BodyMode = string(entity.BodyModeNone)
	}
	return req
}

func ensureLeadingSlash(s string) string {
	if s == "" || strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

func ensureContentTypeHeader(headers []entity.KeyValue, mime string) []entity.KeyValue {
	for _, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h.Key), "Content-Type") {
			return headers
		}
	}
	return append(headers, entity.KeyValue{Key: "Content-Type", Value: mime})
}

// openAPIExampleOrEmpty returns a string example for query/header params.
func openAPIExampleOrEmpty(example, schema json.RawMessage) string {
	if len(example) > 0 {
		return strings.Trim(stringOrEmpty(example), `"`)
	}
	if len(schema) == 0 {
		return ""
	}
	var sch map[string]json.RawMessage
	if err := json.Unmarshal(schema, &sch); err != nil {
		return ""
	}
	if ex, ok := sch["example"]; ok {
		return strings.Trim(stringOrEmpty(ex), `"`)
	}
	if def, ok := sch["default"]; ok {
		return strings.Trim(stringOrEmpty(def), `"`)
	}
	return ""
}

// openAPIExampleAsJSON picks the best available body example from a media type definition.
func openAPIExampleAsJSON(mt openAPIMediaType) string {
	if len(mt.Example) > 0 {
		return prettyJSON(mt.Example)
	}
	for _, ex := range mt.Examples {
		if len(ex.Value) > 0 {
			return prettyJSON(ex.Value)
		}
	}
	return ""
}

// openAPIFormFieldsFromSchema best-effort extracts { "properties": { "a": {...} } } to key list.
func openAPIFormFieldsFromSchema(schema json.RawMessage) map[string]string {
	out := make(map[string]string)
	if len(schema) == 0 {
		return out
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(schema, &obj); err != nil {
		return out
	}
	props, ok := obj["properties"]
	if !ok {
		return out
	}
	var propMap map[string]json.RawMessage
	if err := json.Unmarshal(props, &propMap); err != nil {
		return out
	}
	for k := range propMap {
		out[k] = ""
	}
	return out
}

func prettyJSON(raw json.RawMessage) string {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(b)
}
