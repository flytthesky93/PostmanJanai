package usecase

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// ExportUsecase serializes folder trees to portable formats (Postman, …).
type ExportUsecase interface {
	ExportPostmanV21CollectionJSON(ctx context.Context, rootFolderID string) ([]byte, error)
}

type exportUsecaseImpl struct {
	folders  repository.FolderRepository
	requests repository.RequestRepository
}

func NewExportUsecase(folders repository.FolderRepository, requests repository.RequestRepository) ExportUsecase {
	return &exportUsecaseImpl{folders: folders, requests: requests}
}

func (u *exportUsecaseImpl) ExportPostmanV21CollectionJSON(ctx context.Context, rootFolderID string) ([]byte, error) {
	root, err := u.folders.GetByID(ctx, rootFolderID)
	if err != nil {
		return nil, err
	}
	if root.ParentID != nil {
		return nil, errors.New("export must start from a root folder (collection)")
	}
	items, err := u.postmanItemsForFolder(ctx, rootFolderID)
	if err != nil {
		return nil, err
	}
	doc := map[string]interface{}{
		"info": map[string]interface{}{
			"name":   root.Name,
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		"item": items,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (u *exportUsecaseImpl) postmanItemsForFolder(ctx context.Context, folderID string) ([]interface{}, error) {
	var out []interface{}
	children, err := u.folders.ListChildren(ctx, folderID)
	if err != nil {
		return nil, err
	}
	for _, ch := range children {
		sub, err := u.postmanItemsForFolder(ctx, ch.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"name": ch.Name,
			"item": sub,
		})
	}
	reqs, err := u.requests.ListByFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}
	for _, sum := range reqs {
		full, err := u.requests.GetByID(ctx, sum.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"name":    sum.Name,
			"request": buildPostmanRequestObject(full),
		})
	}
	return out, nil
}

func mergeURLWithQuery(base string, query []entity.KeyValue) string {
	raw := strings.TrimSpace(base)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for _, p := range query {
		k := strings.TrimSpace(p.Key)
		if k != "" {
			q.Set(k, p.Value)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func buildPostmanRequestObject(full *entity.SavedRequestFull) map[string]interface{} {
	if full == nil {
		return map[string]interface{}{}
	}
	method := strings.ToUpper(strings.TrimSpace(full.Method))
	if method == "" {
		method = "GET"
	}
	finalURL := mergeURLWithQuery(full.URL, full.QueryParams)

	var hdrs []interface{}
	for _, h := range full.Headers {
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		hdrs = append(hdrs, map[string]interface{}{
			"key":   k,
			"value": h.Value,
		})
	}

	req := map[string]interface{}{
		"method": method,
		"header": hdrs,
		"url":    finalURL,
	}

	mode := strings.ToLower(strings.TrimSpace(full.BodyMode))
	switch mode {
	case "", "none":
		// no body
	case "raw":
		raw := ""
		if full.RawBody != nil {
			raw = *full.RawBody
		}
		req["body"] = map[string]interface{}{
			"mode": "raw",
			"raw":  raw,
		}
	case "xml":
		raw := ""
		if full.RawBody != nil {
			raw = *full.RawBody
		}
		req["body"] = map[string]interface{}{
			"mode": "raw",
			"raw":  raw,
			"options": map[string]interface{}{
				"raw": map[string]interface{}{"language": "xml"},
			},
		}
	case "form_urlencoded":
		var pairs []interface{}
		for _, f := range full.FormFields {
			k := strings.TrimSpace(f.Key)
			if k == "" {
				continue
			}
			pairs = append(pairs, map[string]interface{}{
				"key": k, "value": f.Value, "type": "text",
			})
		}
		req["body"] = map[string]interface{}{
			"mode":       "urlencoded",
			"urlencoded": pairs,
		}
	case "multipart":
		var parts []interface{}
		for _, p := range full.MultipartParts {
			k := strings.TrimSpace(p.Key)
			if k == "" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(p.Kind), "file") {
				fp := strings.TrimSpace(p.FilePath)
				if fp == "" {
					continue
				}
				parts = append(parts, map[string]interface{}{
					"key": k,
					"type": "file",
					"src":  fp,
				})
			} else {
				parts = append(parts, map[string]interface{}{
					"key": k, "type": "text", "value": p.Value,
				})
			}
		}
		req["body"] = map[string]interface{}{
			"mode":     "formdata",
			"formdata": parts,
		}
	default:
		if full.RawBody != nil && strings.TrimSpace(*full.RawBody) != "" {
			req["body"] = map[string]interface{}{
				"mode": "raw",
				"raw":  *full.RawBody,
			}
		}
	}

	if full.Auth != nil {
		if a := postmanAuthBlock(full.Auth); a != nil {
			req["auth"] = a
		}
	}

	return req
}

func postmanAuthBlock(a *entity.RequestAuth) map[string]interface{} {
	if a == nil {
		return nil
	}
	t := strings.ToLower(strings.TrimSpace(a.Type))
	switch t {
	case "", "none":
		return nil
	case "bearer":
		tok := strings.TrimSpace(a.BearerToken)
		if tok == "" {
			return nil
		}
		return map[string]interface{}{
			"type": "bearer",
			"bearer": []interface{}{
				map[string]interface{}{"key": "token", "value": tok, "type": "string"},
			},
		}
	case "basic":
		return map[string]interface{}{
			"type": "basic",
			"basic": []interface{}{
				map[string]interface{}{"key": "username", "value": strings.TrimSpace(a.Username), "type": "string"},
				map[string]interface{}{"key": "password", "value": a.Password, "type": "string"},
			},
		}
	case "apikey":
		name := strings.TrimSpace(a.APIKeyName)
		val := strings.TrimSpace(a.APIKey)
		if name == "" || val == "" {
			return nil
		}
		where := strings.ToLower(strings.TrimSpace(a.APIKeyIn))
		if where == "" {
			where = "header"
		}
		return map[string]interface{}{
			"type": "apikey",
			"apikey": []interface{}{
				map[string]interface{}{"key": "key", "value": name, "type": "string"},
				map[string]interface{}{"key": "value", "value": val, "type": "string"},
				map[string]interface{}{"key": "in", "value": where, "type": "string"},
			},
		}
	default:
		return nil
	}
}
