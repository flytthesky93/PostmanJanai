package service

import (
	"strings"
	"testing"

	"PostmanJanai/internal/entity"
)

const insomniaSample = `{
  "_type": "export",
  "__export_format": 4,
  "resources": [
    {"_id":"wrk_1","_type":"workspace","name":"MySpace","description":"d"},
    {"_id":"env_1","_type":"environment","parentId":"wrk_1","name":"Base","data":{"baseUrl":"https://a.test","port":8080}},
    {"_id":"fld_1","_type":"request_group","parentId":"wrk_1","name":"Users","metaSortKey":1},
    {
      "_id":"req_1","_type":"request","parentId":"fld_1","name":"List",
      "method":"GET","url":"https://a.test/users",
      "metaSortKey":1,
      "parameters":[{"name":"limit","value":"10"},{"name":"_off","value":"x","disabled":true}],
      "headers":[{"name":"Accept","value":"application/json"}],
      "authentication":{"type":"bearer","token":"xyz"}
    },
    {
      "_id":"req_2","_type":"request","parentId":"fld_1","name":"Create",
      "method":"POST","url":"https://a.test/users",
      "metaSortKey":2,
      "body":{"mimeType":"application/json","text":"{\"name\":\"n\"}"},
      "authentication":{"type":"apikey","key":"X-Key","value":"k","addTo":"header"}
    },
    {
      "_id":"req_3","_type":"request","parentId":"wrk_1","name":"Ping",
      "method":"GET","url":"https://a.test/ping",
      "metaSortKey":0
    }
  ]
}`

func TestImportInsomnia_Tree(t *testing.T) {
	col, err := importInsomnia([]byte(insomniaSample))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if col.Name != "MySpace" || col.FormatLabel != "insomnia_v4" {
		t.Errorf("unexpected meta: %+v", col)
	}
	// Variables sorted alphabetically: baseUrl then port.
	if len(col.Variables) != 2 || col.Variables[0].Key != "baseUrl" || col.Variables[1].Key != "port" {
		t.Errorf("unexpected variables: %+v", col.Variables)
	}
	// Root: Ping (metaSortKey 0) comes before Users (metaSortKey 1).
	if len(col.RootItems) != 2 {
		t.Fatalf("want 2 root items, got %d", len(col.RootItems))
	}
	ping := col.RootItems[0].Request
	if ping == nil || ping.Name != "Ping" {
		t.Fatalf("unexpected first root item: %+v", col.RootItems[0])
	}
	users := col.RootItems[1].Folder
	if users == nil || users.Name != "Users" || len(users.Items) != 2 {
		t.Fatalf("unexpected users folder: %+v", users)
	}
	// Order inside folder by metaSortKey.
	list := users.Items[0].Request
	create := users.Items[1].Request
	if list == nil || list.Name != "List" || len(list.QueryParams) != 1 {
		t.Errorf("unexpected list: %+v", list)
	}
	if list.Auth == nil || list.Auth.Type != "bearer" || list.Auth.BearerToken != "xyz" {
		t.Errorf("bearer auth not imported: %+v", list.Auth)
	}
	if create == nil || create.Name != "Create" {
		t.Fatalf("unexpected create: %+v", create)
	}
	if create.BodyMode != string(entity.BodyModeRaw) || create.RawBody == nil || !strings.Contains(*create.RawBody, "name") {
		t.Errorf("create body not imported: %+v", create)
	}
	if create.Auth == nil || create.Auth.Type != "apikey" || create.Auth.APIKeyIn != "header" || create.Auth.APIKeyName != "X-Key" {
		t.Errorf("apikey auth not imported: %+v", create.Auth)
	}
}

func TestImportInsomnia_Rejects(t *testing.T) {
	if _, err := importInsomnia([]byte(`{"_type":"other","resources":[]}`)); err == nil {
		t.Fatal("should reject non-export")
	}
	if _, err := importInsomnia([]byte(`{"_type":"export","resources":[]}`)); err == nil {
		t.Fatal("should reject empty resources")
	}
}
