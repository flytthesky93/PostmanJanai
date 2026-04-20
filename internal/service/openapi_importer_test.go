package service

import (
	"strings"
	"testing"

	"PostmanJanai/internal/entity"
)

const openAPIJSONSample = `{
  "openapi": "3.0.1",
  "info": { "title": "Petstore", "version": "1.0.0", "description": "demo" },
  "servers": [{ "url": "https://api.petstore.test/v1" }],
  "security": [{ "bearerAuth": [] }],
  "tags": [
    {"name": "pets", "description": "Pet ops"},
    {"name": "users"}
  ],
  "paths": {
    "/pets": {
      "get": {
        "tags": ["pets"],
        "summary": "List pets",
        "parameters": [
          {"name": "limit", "in": "query", "schema": {"type": "integer", "example": 10}},
          {"name": "X-Trace", "in": "header", "schema": {"type": "string"}}
        ]
      },
      "post": {
        "tags": ["pets"],
        "summary": "Create pet",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"type": "object"},
              "example": {"name": "rex", "tag": "dog"}
            }
          }
        }
      }
    },
    "/users/{id}": {
      "get": {
        "tags": ["users"],
        "summary": "Get user",
        "security": [],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}
        ]
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": { "type": "http", "scheme": "bearer" }
    }
  }
}`

func TestImportOpenAPI_JSON(t *testing.T) {
	col, err := importOpenAPI([]byte(openAPIJSONSample))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if col.Name != "Petstore" || col.FormatLabel != "openapi_3.x" {
		t.Errorf("unexpected collection meta: %+v", col)
	}
	// baseUrl variable seeded from server.
	if len(col.Variables) == 0 || col.Variables[0].Key != "baseUrl" || !strings.Contains(col.Variables[0].Value, "petstore.test") {
		t.Errorf("baseUrl variable not seeded: %+v", col.Variables)
	}
	// Two tags → two folders, both with items.
	if len(col.RootItems) != 2 {
		t.Fatalf("want 2 folders, got %d", len(col.RootItems))
	}
	pets := col.RootItems[0].Folder
	if pets == nil || pets.Name != "pets" || len(pets.Items) != 2 {
		t.Fatalf("unexpected pets folder: %+v", pets)
	}
	// List pets: bearer auth inherited from global security.
	list := pets.Items[0].Request
	if list.Auth == nil || list.Auth.Type != "bearer" {
		t.Errorf("list should inherit bearer auth, got %+v", list.Auth)
	}
	if list.URL != "{{baseUrl}}/pets" {
		t.Errorf("unexpected list URL: %q", list.URL)
	}
	// Query example propagated from schema.example.
	foundLimit := false
	for _, q := range list.QueryParams {
		if q.Key == "limit" && q.Value == "10" {
			foundLimit = true
		}
	}
	if !foundLimit {
		t.Errorf("query example not picked up: %+v", list.QueryParams)
	}
	// Create pet: raw JSON body from example.
	create := pets.Items[1].Request
	if create.BodyMode != string(entity.BodyModeRaw) || create.RawBody == nil || !strings.Contains(*create.RawBody, "rex") {
		t.Errorf("create body not imported: %+v", create)
	}
	ct := ""
	for _, h := range create.Headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			ct = h.Value
		}
	}
	if ct != "application/json" {
		t.Errorf("expected Content-Type json, got %q", ct)
	}

	// Get user: security: [] → no auth
	users := col.RootItems[1].Folder
	getUser := users.Items[0].Request
	if getUser.Auth == nil || getUser.Auth.Type != "none" {
		t.Errorf("security: [] should mean no auth, got %+v", getUser.Auth)
	}
	if getUser.URL != "{{baseUrl}}/users/{id}" {
		t.Errorf("unexpected user URL: %q", getUser.URL)
	}
}

const openAPIYAMLSample = `
openapi: 3.0.0
info:
  title: Tiny
  version: "1.0"
servers:
  - url: https://t.test
paths:
  /ping:
    get:
      summary: Ping
      responses:
        "200":
          description: ok
`

func TestImportOpenAPI_YAML(t *testing.T) {
	col, err := importOpenAPI([]byte(openAPIYAMLSample))
	if err != nil {
		t.Fatalf("yaml import: %v", err)
	}
	if col.Name != "Tiny" {
		t.Errorf("want name Tiny, got %q", col.Name)
	}
	if len(col.RootItems) != 1 {
		t.Fatalf("want 1 root request, got %d", len(col.RootItems))
	}
	ping := col.RootItems[0].Request
	if ping == nil || ping.Method != "GET" || ping.URL != "{{baseUrl}}/ping" {
		t.Errorf("unexpected yaml request: %+v", ping)
	}
}

func TestImportOpenAPI_Rejects(t *testing.T) {
	if _, err := importOpenAPI([]byte(`{"swagger":"2.0","info":{"title":"legacy"}}`)); err == nil {
		t.Errorf("swagger 2.0 should be rejected")
	}
	if _, err := importOpenAPI([]byte(`{"openapi":"3.0","info":{"title":"none"},"paths":{}}`)); err == nil {
		t.Errorf("empty paths should be rejected")
	}
}
