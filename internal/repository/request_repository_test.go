package repository

import (
	"context"
	"testing"

	"PostmanJanai/ent"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/testutil"
)

func newRequestTestRig(t *testing.T) (context.Context, *ent.Client, FolderRepository, RequestRepository) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	return ctx, client, NewFolderRepository(client), NewRequestRepository(client)
}

func TestRequestRepo_CreateFullAndGetByIDRoundTrip(t *testing.T) {
	ctx, _, folders, reqs := newRequestTestRig(t)
	folderID, _ := folders.Create(ctx, &entity.FolderItem{Name: "F"})

	raw := `{"hello":"world"}`
	in := &entity.SavedRequestFull{
		FolderID: folderID,
		Name:     "Req A",
		Method:   "POST",
		URL:      "https://api.example.com/items",
		BodyMode: "raw",
		RawBody:  &raw,
		Headers: []entity.KeyValue{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "X-Trace", Value: "1"},
		},
		QueryParams: []entity.KeyValue{
			{Key: "limit", Value: "10"},
		},
		Auth: &entity.RequestAuth{Type: "bearer", BearerToken: "abc"},
	}
	id, err := reqs.CreateFull(ctx, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := reqs.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Req A" || got.Method != "POST" || got.URL != "https://api.example.com/items" {
		t.Fatalf("metadata mismatch: %+v", got)
	}
	if got.BodyMode != "raw" || got.RawBody == nil || *got.RawBody != raw {
		t.Fatalf("body mismatch: mode=%s raw=%v", got.BodyMode, got.RawBody)
	}
	if len(got.Headers) != 2 || got.Headers[0].Key != "Content-Type" || got.Headers[1].Key != "X-Trace" {
		t.Fatalf("headers mismatch: %+v", got.Headers)
	}
	if len(got.QueryParams) != 1 || got.QueryParams[0].Key != "limit" {
		t.Fatalf("query mismatch: %+v", got.QueryParams)
	}
	if got.Auth == nil || got.Auth.Type != "bearer" || got.Auth.BearerToken != "abc" {
		t.Fatalf("auth mismatch: %+v", got.Auth)
	}
}

func TestRequestRepo_CreateFullWithFormAndMultipart(t *testing.T) {
	ctx, _, folders, reqs := newRequestTestRig(t)
	folderID, _ := folders.Create(ctx, &entity.FolderItem{Name: "F"})

	in := &entity.SavedRequestFull{
		FolderID: folderID,
		Name:     "Form",
		Method:   "POST",
		URL:      "https://example.com",
		BodyMode: "form_urlencoded",
		FormFields: []entity.KeyValue{
			{Key: "username", Value: "alice"},
			{Key: "password", Value: "secret"},
		},
		MultipartParts: []entity.MultipartPart{
			{Key: "caption", Kind: "text", Value: "hi"},
			{Key: "attachment", Kind: "file", FilePath: "/tmp/test.bin"},
		},
	}
	id, err := reqs.CreateFull(ctx, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := reqs.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.FormFields) != 2 {
		t.Fatalf("form fields want 2 got %d: %+v", len(got.FormFields), got.FormFields)
	}
	if got.FormFields[0].Key != "username" || got.FormFields[0].Value != "alice" {
		t.Fatalf("form[0] = %+v", got.FormFields[0])
	}
	if len(got.MultipartParts) != 2 {
		t.Fatalf("multipart parts want 2 got %d: %+v", len(got.MultipartParts), got.MultipartParts)
	}
	if got.MultipartParts[1].Kind != "file" || got.MultipartParts[1].FilePath != "/tmp/test.bin" {
		t.Fatalf("multipart file row wrong: %+v", got.MultipartParts[1])
	}
}

func TestRequestRepo_UpdateFullReplacesChildRows(t *testing.T) {
	ctx, _, folders, reqs := newRequestTestRig(t)
	folderID, _ := folders.Create(ctx, &entity.FolderItem{Name: "F"})

	id, err := reqs.CreateFull(ctx, &entity.SavedRequestFull{
		FolderID: folderID,
		Name:     "R",
		Method:   "GET",
		URL:      "https://e.com",
		BodyMode: "none",
		Headers: []entity.KeyValue{
			{Key: "A", Value: "1"},
			{Key: "B", Value: "2"},
			{Key: "C", Value: "3"},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Update: single header replaces all, add query param, switch to raw body.
	raw := "xyz"
	err = reqs.UpdateFull(ctx, &entity.SavedRequestFull{
		ID:          id,
		FolderID:    folderID,
		Name:        "R",
		Method:      "POST",
		URL:         "https://e.com/2",
		BodyMode:    "raw",
		RawBody:     &raw,
		Headers:     []entity.KeyValue{{Key: "Only", Value: "one"}},
		QueryParams: []entity.KeyValue{{Key: "q", Value: "v"}},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := reqs.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Method != "POST" || got.URL != "https://e.com/2" {
		t.Fatalf("metadata not updated: %+v", got)
	}
	if len(got.Headers) != 1 || got.Headers[0].Key != "Only" {
		t.Fatalf("headers not replaced: %+v", got.Headers)
	}
	if len(got.QueryParams) != 1 || got.QueryParams[0].Key != "q" {
		t.Fatalf("query not added: %+v", got.QueryParams)
	}
	if got.RawBody == nil || *got.RawBody != raw {
		t.Fatalf("raw body not set: %v", got.RawBody)
	}
}

func TestRequestRepo_ExistsNameInFolderAndMove(t *testing.T) {
	ctx, _, folders, reqs := newRequestTestRig(t)
	f1, _ := folders.Create(ctx, &entity.FolderItem{Name: "F1"})
	f2, _ := folders.Create(ctx, &entity.FolderItem{Name: "F2"})

	id1, _ := reqs.CreateFull(ctx, &entity.SavedRequestFull{
		FolderID: f1, Name: "Req", Method: "GET", URL: "https://e.com", BodyMode: "none",
	})

	exists, _ := reqs.ExistsNameInFolder(ctx, f1, "Req", nil)
	if !exists {
		t.Fatal("expected Req to exist in F1")
	}
	exists, _ = reqs.ExistsNameInFolder(ctx, f2, "Req", nil)
	if exists {
		t.Fatal("Req should not exist in F2")
	}
	exists, _ = reqs.ExistsNameInFolder(ctx, f1, "Req", &id1)
	if exists {
		t.Fatal("excluding self should report not-exist")
	}

	if err := reqs.MoveToFolder(ctx, id1, f2); err != nil {
		t.Fatalf("move: %v", err)
	}
	got, _ := reqs.GetByID(ctx, id1)
	if got.FolderID != f2 {
		t.Fatalf("move did not change folder_id, got %s", got.FolderID)
	}
}

func TestRequestRepo_SearchByNameOrURL(t *testing.T) {
	ctx, _, folders, reqs := newRequestTestRig(t)
	folderID, _ := folders.Create(ctx, &entity.FolderItem{Name: "F"})

	_, _ = reqs.CreateFull(ctx, &entity.SavedRequestFull{FolderID: folderID, Name: "List Users", Method: "GET", URL: "https://api.example.com/users", BodyMode: "none"})
	_, _ = reqs.CreateFull(ctx, &entity.SavedRequestFull{FolderID: folderID, Name: "Create User", Method: "POST", URL: "https://api.example.com/users", BodyMode: "none"})
	_, _ = reqs.CreateFull(ctx, &entity.SavedRequestFull{FolderID: folderID, Name: "Ping", Method: "GET", URL: "https://health.example.com/ping", BodyMode: "none"})

	// Match by name.
	hits, truncated, err := reqs.SearchByNameOrURL(ctx, "user", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if truncated || len(hits) != 2 {
		t.Fatalf("want 2 hits truncated=false, got %d truncated=%v", len(hits), truncated)
	}
	// Match by URL.
	hits, _, _ = reqs.SearchByNameOrURL(ctx, "health", 10)
	if len(hits) != 1 || hits[0].Name != "Ping" {
		t.Fatalf("health search failed: %+v", hits)
	}
	// Truncation.
	hits, truncated, _ = reqs.SearchByNameOrURL(ctx, "api", 1)
	if !truncated || len(hits) != 1 {
		t.Fatalf("want truncated hits=1, got truncated=%v len=%d", truncated, len(hits))
	}
	// Empty query returns nil.
	hits, truncated, err = reqs.SearchByNameOrURL(ctx, "   ", 10)
	if err != nil || hits != nil || truncated {
		t.Fatalf("empty query: hits=%v truncated=%v err=%v", hits, truncated, err)
	}
}

func TestRequestRepo_DeleteCleansUpChildRows(t *testing.T) {
	ctx, client, folders, reqs := newRequestTestRig(t)
	folderID, _ := folders.Create(ctx, &entity.FolderItem{Name: "F"})
	id, _ := reqs.CreateFull(ctx, &entity.SavedRequestFull{
		FolderID:    folderID,
		Name:        "R",
		Method:      "GET",
		URL:         "https://e.com",
		BodyMode:    "none",
		Headers:     []entity.KeyValue{{Key: "A", Value: "1"}},
		QueryParams: []entity.KeyValue{{Key: "q", Value: "1"}},
		FormFields:  []entity.KeyValue{{Key: "f", Value: "v"}},
	})

	if err := reqs.DeleteByID(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if n, _ := client.Request.Query().Count(ctx); n != 0 {
		t.Fatalf("request row not deleted, count=%d", n)
	}
	if n, _ := client.RequestHeader.Query().Count(ctx); n != 0 {
		t.Fatalf("headers not deleted, count=%d", n)
	}
	if n, _ := client.RequestQueryParam.Query().Count(ctx); n != 0 {
		t.Fatalf("query params not deleted, count=%d", n)
	}
	if n, _ := client.RequestFormField.Query().Count(ctx); n != 0 {
		t.Fatalf("form fields not deleted, count=%d", n)
	}
}
