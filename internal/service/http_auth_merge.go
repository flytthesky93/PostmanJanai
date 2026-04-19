package service

import (
	"PostmanJanai/internal/entity"
	"encoding/base64"
	"strings"
)

// MergeAuthIntoHeadersAndQuery applies resolved Auth to headers and query params (mutates in).
// Runs after SubstituteEnvVars. Bearer/Basic replace any existing Authorization header.
func MergeAuthIntoHeadersAndQuery(in *entity.HTTPExecuteInput) {
	if in == nil || in.Auth == nil {
		return
	}
	t := strings.ToLower(strings.TrimSpace(in.Auth.Type))
	if t == "" || t == "none" {
		return
	}

	switch t {
	case "bearer":
		tok := strings.TrimSpace(in.Auth.BearerToken)
		if tok == "" {
			return
		}
		removeHeaderCI(&in.Headers, "authorization")
		in.Headers = append(in.Headers, entity.KeyValue{Key: "Authorization", Value: "Bearer " + tok})

	case "basic":
		user := strings.TrimSpace(in.Auth.Username)
		pass := in.Auth.Password
		if user == "" && pass == "" {
			return
		}
		raw := user + ":" + pass
		b64 := base64.StdEncoding.EncodeToString([]byte(raw))
		removeHeaderCI(&in.Headers, "authorization")
		in.Headers = append(in.Headers, entity.KeyValue{Key: "Authorization", Value: "Basic " + b64})

	case "apikey":
		keyName := strings.TrimSpace(in.Auth.APIKeyName)
		keyVal := strings.TrimSpace(in.Auth.APIKey)
		if keyName == "" || keyVal == "" {
			return
		}
		where := strings.ToLower(strings.TrimSpace(in.Auth.APIKeyIn))
		if where == "" {
			where = "header"
		}
		if where == "query" {
			removeQueryParamCI(&in.QueryParams, keyName)
			in.QueryParams = append(in.QueryParams, entity.KeyValue{Key: keyName, Value: keyVal})
		} else {
			removeHeaderCI(&in.Headers, keyName)
			in.Headers = append(in.Headers, entity.KeyValue{Key: keyName, Value: keyVal})
		}
	}
}

func removeHeaderCI(headers *[]entity.KeyValue, name string) {
	if headers == nil || name == "" {
		return
	}
	nn := strings.ToLower(strings.TrimSpace(name))
	out := (*headers)[:0]
	for _, h := range *headers {
		if strings.ToLower(strings.TrimSpace(h.Key)) == nn {
			continue
		}
		out = append(out, h)
	}
	*headers = out
}

func removeQueryParamCI(params *[]entity.KeyValue, name string) {
	if params == nil || name == "" {
		return
	}
	nn := strings.ToLower(strings.TrimSpace(name))
	out := (*params)[:0]
	for _, p := range *params {
		if strings.ToLower(strings.TrimSpace(p.Key)) == nn {
			continue
		}
		out = append(out, p)
	}
	*params = out
}
