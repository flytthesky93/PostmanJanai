package service

import (
	"PostmanJanai/internal/entity"
	"errors"
	"net/url"
	"strings"
)

// FinalURLForRequest returns the URL after merging query params, matching
// HTTPExecutor.Execute (scheme/host required).
func FinalURLForRequest(in *entity.HTTPExecuteInput) (string, error) {
	if in == nil {
		return "", errors.New("nil input")
	}
	raw := strings.TrimSpace(in.URL)
	if raw == "" {
		return "", errors.New("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errors.New("URL must include scheme and host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("only http and https are supported")
	}
	q := u.Query()
	for _, p := range in.QueryParams {
		k := strings.TrimSpace(p.Key)
		if k != "" {
			q.Add(k, p.Value)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
