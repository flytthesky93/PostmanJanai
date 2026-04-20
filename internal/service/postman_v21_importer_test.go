package service

import (
	"strings"
	"testing"

	"PostmanJanai/internal/entity"
)

const postmanV21Sample = `{
  "info": {
    "name": "Sample API",
    "description": "Demo collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Users",
      "item": [
        {
          "name": "List users",
          "request": {
            "method": "GET",
            "header": [
              {"key": "Accept", "value": "application/json"},
              {"key": "X-Ignore", "value": "1", "disabled": true}
            ],
            "url": {
              "raw": "https://api.example.com/users?limit=10",
              "protocol": "https",
              "host": ["api","example","com"],
              "path": ["users"],
              "query": [{"key":"limit","value":"10"}]
            }
          }
        },
        {
          "name": "Create user",
          "request": {
            "method": "POST",
            "header": [{"key":"Content-Type","value":"application/json"}],
            "url": "https://api.example.com/users",
            "body": {
              "mode": "raw",
              "raw": "{\"name\":\"{{user_name}}\"}",
              "options": {"raw": {"language": "json"}}
            },
            "auth": {
              "type": "bearer",
              "bearer": [{"key":"token","value":"{{token}}","type":"string"}]
            }
          }
        }
      ]
    },
    {
      "name": "Ping",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/ping"
      }
    }
  ],
  "variable": [
    {"key": "baseUrl", "value": "https://api.example.com"},
    {"key": "token",   "value": ""}
  ],
  "auth": {
    "type": "apikey",
    "apikey": [
      {"key":"key","value":"X-API-Key"},
      {"key":"value","value":"{{apiKey}}"},
      {"key":"in","value":"header"}
    ]
  }
}`

func TestImportPostmanV21_Basic(t *testing.T) {
	col, err := importPostmanV21([]byte(postmanV21Sample))
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if col.Name != "Sample API" {
		t.Errorf("want root name 'Sample API', got %q", col.Name)
	}
	if col.FormatLabel != "postman_v2.1" {
		t.Errorf("want format postman_v2.1, got %q", col.FormatLabel)
	}
	if len(col.Variables) != 2 {
		t.Errorf("want 2 variables, got %d", len(col.Variables))
	}
	if len(col.RootItems) != 2 {
		t.Fatalf("want 2 root items, got %d", len(col.RootItems))
	}

	// Folder "Users" should contain 2 requests.
	users := col.RootItems[0].Folder
	if users == nil {
		t.Fatalf("first root item should be a folder")
	}
	if users.Name != "Users" || len(users.Items) != 2 {
		t.Fatalf("unexpected folder shape: name=%q items=%d", users.Name, len(users.Items))
	}

	// List users: collection-level apikey auth should cascade down when request has none.
	list := users.Items[0].Request
	if list == nil {
		t.Fatalf("want 'List users' request")
	}
	if list.Method != "GET" || list.URL == "" {
		t.Errorf("unexpected list request: method=%q url=%q", list.Method, list.URL)
	}
	// Disabled header filtered out.
	for _, h := range list.Headers {
		if strings.EqualFold(h.Key, "X-Ignore") {
			t.Errorf("disabled header leaked through: %+v", h)
		}
	}
	if list.Auth == nil || list.Auth.Type != "apikey" || list.Auth.APIKeyName != "X-API-Key" {
		t.Errorf("expected inherited apikey auth, got %+v", list.Auth)
	}

	// Create user: own bearer auth overrides parent apikey.
	create := users.Items[1].Request
	if create == nil || create.Method != "POST" {
		t.Fatalf("want Create user POST, got %+v", create)
	}
	if create.Auth == nil || create.Auth.Type != "bearer" || !strings.Contains(create.Auth.BearerToken, "{{token}}") {
		t.Errorf("expected bearer auth with {{token}}, got %+v", create.Auth)
	}
	if create.BodyMode != string(entity.BodyModeRaw) || create.RawBody == nil || !strings.Contains(*create.RawBody, "{{user_name}}") {
		t.Errorf("expected raw JSON body with {{user_name}}, got mode=%q body=%v", create.BodyMode, create.RawBody)
	}

	// Top-level Ping request (no folder wrapper).
	ping := col.RootItems[1].Request
	if ping == nil || ping.Method != "GET" || !strings.HasSuffix(ping.URL, "/ping") {
		t.Errorf("unexpected ping: %+v", ping)
	}
}

func TestImportPostmanV21_RejectsEmpty(t *testing.T) {
	if _, err := importPostmanV21([]byte(`{}`)); err == nil {
		t.Fatal("expected error for empty document")
	}
	if _, err := importPostmanV21([]byte(`{"info":{"name":"x"},"item":[]}`)); err == nil {
		t.Fatal("expected error for collection without items")
	}
}

func TestImportPostmanV21_UrlEncodedBody(t *testing.T) {
	doc := `{
      "info":{"name":"x"},
      "item":[{
        "name":"Form",
        "request":{
          "method":"POST",
          "url":"https://x.test/form",
          "body":{
            "mode":"urlencoded",
            "urlencoded":[
              {"key":"a","value":"1"},
              {"key":"b","value":"2","disabled":true}
            ]
          }
        }
      }]
    }`
	col, err := importPostmanV21([]byte(doc))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	r := col.RootItems[0].Request
	if r.BodyMode != string(entity.BodyModeFormURLEncoded) || len(r.FormFields) != 1 || r.FormFields[0].Key != "a" {
		t.Fatalf("unexpected form fields: %+v", r)
	}
}
