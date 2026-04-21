package repository

import (
	"context"
	"testing"
	"time"

	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/testutil"
)

func TestHistoryRepo_SaveAndListWithFilter(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	folders := NewFolderRepository(client)
	history := NewHistoryRepository(client)

	rootA, _ := folders.Create(ctx, &entity.FolderItem{Name: "A"})
	rootB, _ := folders.Create(ctx, &entity.FolderItem{Name: "B"})

	reqBody := `{"q":"x"}`
	respBody := `{"ok":true}`
	now := time.Now().UTC()
	dur := 42
	size := 123

	for i := 0; i < 3; i++ {
		item := &entity.HistoryItem{
			RootFolderID:      &rootA,
			Method:            "GET",
			URL:               "https://a.example.com/x",
			StatusCode:        200,
			DurationMs:        &dur,
			ResponseSizeBytes: &size,
			RequestBody:       &reqBody,
			ResponseBody:      &respBody,
			CreatedAt:         now.Add(time.Duration(i) * time.Second),
		}
		if err := history.Save(ctx, item); err != nil {
			t.Fatalf("save[%d]: %v", i, err)
		}
	}
	if err := history.Save(ctx, &entity.HistoryItem{
		RootFolderID: &rootB,
		Method:       "POST",
		URL:          "https://b.example.com/y",
		StatusCode:   500,
		CreatedAt:    now.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("save b: %v", err)
	}

	// No filter → 4 rows, ordered newest first.
	all, err := history.ListSummaries(ctx, nil)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("want 4 rows, got %d", len(all))
	}
	// Newest should be rootB (offset +5s) → URL y.
	if all[0].URL != "https://b.example.com/y" {
		t.Fatalf("order wrong, newest URL %q", all[0].URL)
	}

	// Filter by rootA → 3 rows.
	aHits, err := history.ListSummaries(ctx, &rootA)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if len(aHits) != 3 {
		t.Fatalf("filter A want 3, got %d", len(aHits))
	}
	for _, h := range aHits {
		if h.RootFolderID == nil || *h.RootFolderID != rootA {
			t.Fatalf("row has wrong root_folder_id: %+v", h)
		}
	}

	// Get by id returns full body snapshot.
	got, err := history.GetByID(ctx, aHits[0].ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ResponseBody == nil || *got.ResponseBody != respBody {
		t.Fatalf("response body not persisted: %v", got.ResponseBody)
	}
	if got.RequestBody == nil || *got.RequestBody != reqBody {
		t.Fatalf("request body not persisted: %v", got.RequestBody)
	}
}

func TestHistoryRepo_DeleteByID(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	history := NewHistoryRepository(client)

	if err := history.Save(ctx, &entity.HistoryItem{
		Method: "GET", URL: "https://e.com", StatusCode: 200,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	list, _ := history.ListSummaries(ctx, nil)
	if len(list) != 1 {
		t.Fatalf("expected 1 row, got %d", len(list))
	}
	if err := history.DeleteByID(ctx, list[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = history.ListSummaries(ctx, nil)
	if len(list) != 0 {
		t.Fatalf("row should be gone, got %d", len(list))
	}
}
