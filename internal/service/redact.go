package service

import (
	"PostmanJanai/internal/entity"
	"sort"
	"strings"
)

const redactMarker = "***"

// RedactSecretStrings replaces occurrences of any non-empty secret string in text-like fields.
// Matching is case-sensitive substring replace (longest secrets first to reduce partial overlaps).
func RedactSecretStrings(s string, secrets []string) string {
	if s == "" || len(secrets) == 0 {
		return s
	}
	uniq := make(map[string]struct{})
	var list []string
	for _, sec := range secrets {
		t := strings.TrimSpace(sec)
		if t == "" {
			continue
		}
		if _, ok := uniq[t]; ok {
			continue
		}
		uniq[t] = struct{}{}
		list = append(list, t)
	}
	if len(list) == 0 {
		return s
	}
	sort.Slice(list, func(i, j int) bool { return len(list[i]) > len(list[j]) })

	out := s
	for _, sec := range list {
		if strings.Contains(out, sec) {
			out = strings.ReplaceAll(out, sec, redactMarker)
		}
	}
	return out
}

// RedactHTTPExecuteInput returns a shallow copy of `in` with secrets redacted from URL/body/params.
func RedactHTTPExecuteInput(in *entity.HTTPExecuteInput, secrets []string) *entity.HTTPExecuteInput {
	if in == nil || len(secrets) == 0 {
		return in
	}
	cp := *in
	cp.URL = RedactSecretStrings(cp.URL, secrets)
	cp.Body = RedactSecretStrings(cp.Body, secrets)
	if len(cp.Headers) > 0 {
		cp.Headers = cloneAndRedactKeyValues(cp.Headers, secrets)
	}
	if len(cp.QueryParams) > 0 {
		cp.QueryParams = cloneAndRedactKeyValues(cp.QueryParams, secrets)
	}
	if len(cp.FormFields) > 0 {
		cp.FormFields = cloneAndRedactKeyValues(cp.FormFields, secrets)
	}
	if len(cp.MultipartParts) > 0 {
		mp := make([]entity.MultipartPart, len(cp.MultipartParts))
		for i, p := range cp.MultipartParts {
			mp[i] = p
			mp[i].Value = RedactSecretStrings(p.Value, secrets)
			mp[i].FilePath = RedactSecretStrings(p.FilePath, secrets)
		}
		cp.MultipartParts = mp
	}
	if cp.Auth != nil {
		a := *cp.Auth
		switch strings.ToLower(strings.TrimSpace(a.Type)) {
		case "bearer":
			a.BearerToken = RedactSecretStrings(a.BearerToken, secrets)
		case "basic":
			a.Username = RedactSecretStrings(a.Username, secrets)
			a.Password = RedactSecretStrings(a.Password, secrets)
		case "apikey":
			a.APIKey = RedactSecretStrings(a.APIKey, secrets)
			a.APIKeyName = RedactSecretStrings(a.APIKeyName, secrets)
		}
		cp.Auth = &a
	}
	return &cp
}

func cloneAndRedactKeyValues(in []entity.KeyValue, secrets []string) []entity.KeyValue {
	out := make([]entity.KeyValue, len(in))
	for i, kv := range in {
		out[i] = entity.KeyValue{
			Key:   RedactSecretStrings(kv.Key, secrets),
			Value: RedactSecretStrings(kv.Value, secrets),
		}
	}
	return out
}
