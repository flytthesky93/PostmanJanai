package service

import (
	"testing"

	"PostmanJanai/internal/entity"
)

const postmanV20Sample = `{
  "id": "abc",
  "name": "Legacy API",
  "description": "v2.0 collection",
  "order": ["req_top"],
  "folders": [
    {"id":"fld_users","name":"Users","order":["req_list","req_create"],"folders_order":[]}
  ],
  "requests": [
    {"id":"req_top","name":"Ping","method":"GET","url":"https://x.test/ping"},
    {
      "id":"req_list","name":"List",
      "method":"GET","url":"https://x.test/users",
      "folder":"fld_users",
      "headerData":[{"key":"Accept","value":"application/json"}],
      "queryParams":[{"key":"limit","value":"20","enabled":true}]
    },
    {
      "id":"req_create","name":"Create",
      "method":"POST","url":"https://x.test/users",
      "folder":"fld_users",
      "headers":"Content-Type: application/json\n",
      "dataMode":"raw",
      "rawModeData":"{\"name\":\"bob\"}"
    }
  ],
  "variables": [{"key":"baseUrl","value":"https://x.test"}]
}`

func TestImportPostmanV20_Tree(t *testing.T) {
	col, err := importPostmanV20([]byte(postmanV20Sample))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if col.Name != "Legacy API" || col.FormatLabel != "postman_v2.0" {
		t.Errorf("unexpected collection meta: %+v", col)
	}
	if len(col.Variables) != 1 || col.Variables[0].Key != "baseUrl" {
		t.Errorf("unexpected variables: %+v", col.Variables)
	}
	if len(col.RootItems) != 2 {
		t.Fatalf("want 2 root items (Users folder + Ping), got %d", len(col.RootItems))
	}
	users := col.RootItems[0].Folder
	if users == nil || users.Name != "Users" || len(users.Items) != 2 {
		t.Fatalf("unexpected Users folder: %+v", users)
	}
	list := users.Items[0].Request
	if list == nil || list.Method != "GET" || len(list.QueryParams) != 1 {
		t.Errorf("unexpected list request: %+v", list)
	}
	create := users.Items[1].Request
	if create == nil || create.Method != "POST" {
		t.Fatalf("unexpected create request: %+v", create)
	}
	if create.BodyMode != string(entity.BodyModeRaw) || create.RawBody == nil {
		t.Errorf("expected raw body on Create")
	}
	// Legacy blob headers parsed into structured headers.
	foundCT := false
	for _, h := range create.Headers {
		if h.Key == "Content-Type" && h.Value == "application/json" {
			foundCT = true
		}
	}
	if !foundCT {
		t.Errorf("legacy headers blob not parsed: %+v", create.Headers)
	}
	ping := col.RootItems[1].Request
	if ping == nil || ping.Name != "Ping" {
		t.Errorf("unexpected ping: %+v", ping)
	}
}
