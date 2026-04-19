package service

import (
	"PostmanJanai/internal/entity"
	"regexp"
	"strings"
)

// envVarPlaceholder matches Postman-style {{ variable_name }} (no nested braces inside).
var envVarPlaceholder = regexp.MustCompile(`\{\{\s*([^{}]*?)\s*\}\}`)

// SubstituteEnvVars replaces every {{ key }} in s using vars; unknown or empty keys become "".
func SubstituteEnvVars(s string, vars map[string]string) string {
	if s == "" {
		return s
	}
	if vars == nil {
		vars = map[string]string{}
	}
	return envVarPlaceholder.ReplaceAllStringFunc(s, func(match string) string {
		sub := envVarPlaceholder.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		key := strings.TrimSpace(sub[1])
		if key == "" {
			return ""
		}
		v, ok := vars[key]
		if !ok {
			return ""
		}
		return v
	})
}

// CloneSubstituteHTTPExecuteInput returns a deep copy of in with all substitutable strings resolved.
func CloneSubstituteHTTPExecuteInput(in *entity.HTTPExecuteInput, vars map[string]string) *entity.HTTPExecuteInput {
	if in == nil {
		return nil
	}
	if vars == nil {
		vars = map[string]string{}
	}
	out := *in
	out.URL = SubstituteEnvVars(in.URL, vars)
	out.Body = SubstituteEnvVars(in.Body, vars)

	if len(in.Headers) > 0 {
		out.Headers = make([]entity.KeyValue, len(in.Headers))
		for i, kv := range in.Headers {
			out.Headers[i] = entity.KeyValue{
				Key:   SubstituteEnvVars(kv.Key, vars),
				Value: SubstituteEnvVars(kv.Value, vars),
			}
		}
	}
	if len(in.QueryParams) > 0 {
		out.QueryParams = make([]entity.KeyValue, len(in.QueryParams))
		for i, kv := range in.QueryParams {
			out.QueryParams[i] = entity.KeyValue{
				Key:   SubstituteEnvVars(kv.Key, vars),
				Value: SubstituteEnvVars(kv.Value, vars),
			}
		}
	}
	if len(in.FormFields) > 0 {
		out.FormFields = make([]entity.KeyValue, len(in.FormFields))
		for i, kv := range in.FormFields {
			out.FormFields[i] = entity.KeyValue{
				Key:   SubstituteEnvVars(kv.Key, vars),
				Value: SubstituteEnvVars(kv.Value, vars),
			}
		}
	}
	if len(in.MultipartParts) > 0 {
		out.MultipartParts = make([]entity.MultipartPart, len(in.MultipartParts))
		for i, p := range in.MultipartParts {
			kind := strings.ToLower(strings.TrimSpace(p.Kind))
			out.MultipartParts[i] = entity.MultipartPart{
				Key:  SubstituteEnvVars(p.Key, vars),
				Kind: p.Kind,
			}
			if kind == "file" {
				out.MultipartParts[i].FilePath = SubstituteEnvVars(p.FilePath, vars)
			} else {
				out.MultipartParts[i].Value = SubstituteEnvVars(p.Value, vars)
			}
		}
	}
	if in.Auth != nil {
		a := *in.Auth
		a.Type = SubstituteEnvVars(a.Type, vars)
		a.BearerToken = SubstituteEnvVars(a.BearerToken, vars)
		a.Username = SubstituteEnvVars(a.Username, vars)
		a.Password = SubstituteEnvVars(a.Password, vars)
		a.APIKey = SubstituteEnvVars(a.APIKey, vars)
		a.APIKeyName = SubstituteEnvVars(a.APIKeyName, vars)
		a.APIKeyIn = SubstituteEnvVars(a.APIKeyIn, vars)
		out.Auth = &a
	}
	return &out
}
