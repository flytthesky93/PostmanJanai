package repository

import (
	"context"
	"testing"

	"PostmanJanai/ent"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/testutil"
)

func strPtr(s string) *string { return &s }

func newFolderTestRig(t *testing.T) (context.Context, *ent.Client, FolderRepository) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	return ctx, client, NewFolderRepository(client)
}

func TestFolderRepo_CreateRootAndNested(t *testing.T) {
	ctx, _, repo := newFolderTestRig(t)

	rootID, err := repo.Create(ctx, &entity.FolderItem{Name: "Root"})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	if _, err := repo.Create(ctx, &entity.FolderItem{Name: "Child", ParentID: &rootID}); err != nil {
		t.Fatalf("create child: %v", err)
	}

	roots, err := repo.ListRoots(ctx)
	if err != nil {
		t.Fatalf("list roots: %v", err)
	}
	if len(roots) != 1 || roots[0].Name != "Root" {
		t.Fatalf("unexpected roots: %+v", roots)
	}

	kids, err := repo.ListChildren(ctx, rootID)
	if err != nil {
		t.Fatalf("list children: %v", err)
	}
	if len(kids) != 1 || kids[0].Name != "Child" {
		t.Fatalf("unexpected children: %+v", kids)
	}
}

func TestFolderRepo_RootAndChildNameTaken(t *testing.T) {
	ctx, _, repo := newFolderTestRig(t)

	rootID, err := repo.Create(ctx, &entity.FolderItem{Name: "Root"})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	taken, err := repo.RootNameTaken(ctx, "Root", nil)
	if err != nil || !taken {
		t.Fatalf("RootNameTaken want true, got %v, err %v", taken, err)
	}
	taken, err = repo.RootNameTaken(ctx, "Root", &rootID)
	if err != nil || taken {
		t.Fatalf("RootNameTaken excluding self want false, got %v, err %v", taken, err)
	}
	taken, err = repo.RootNameTaken(ctx, "Fresh", nil)
	if err != nil || taken {
		t.Fatalf("RootNameTaken fresh want false, got %v, err %v", taken, err)
	}

	childID, err := repo.Create(ctx, &entity.FolderItem{Name: "Child", ParentID: &rootID})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	taken, err = repo.ChildNameTaken(ctx, rootID, "Child", nil)
	if err != nil || !taken {
		t.Fatalf("ChildNameTaken want true, got %v err %v", taken, err)
	}
	taken, err = repo.ChildNameTaken(ctx, rootID, "Child", &childID)
	if err != nil || taken {
		t.Fatalf("ChildNameTaken excluding self want false, got %v err %v", taken, err)
	}
}

func TestFolderRepo_UniqueParentIDNameRejectsDuplicate(t *testing.T) {
	ctx, _, repo := newFolderTestRig(t)
	rootID, _ := repo.Create(ctx, &entity.FolderItem{Name: "Root"})
	if _, err := repo.Create(ctx, &entity.FolderItem{Name: "Dup", ParentID: &rootID}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := repo.Create(ctx, &entity.FolderItem{Name: "Dup", ParentID: &rootID}); err == nil {
		t.Fatal("expected UNIQUE(parent_id,name) violation")
	}
}

func TestFolderRepo_DeleteByIDRecursiveAndHistoryFKCleared(t *testing.T) {
	ctx, client, repo := newFolderTestRig(t)

	rootID, err := repo.Create(ctx, &entity.FolderItem{Name: "Root"})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	childID, err := repo.Create(ctx, &entity.FolderItem{Name: "Child", ParentID: &rootID})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	grandID, err := repo.Create(ctx, &entity.FolderItem{Name: "Grand", ParentID: &childID})
	if err != nil {
		t.Fatalf("create grandchild: %v", err)
	}
	_ = grandID

	// Attach a saved request to the deepest folder.
	reqRepo := NewRequestRepository(client)
	reqID, err := reqRepo.CreateFull(ctx, &entity.SavedRequestFull{
		FolderID: grandID,
		Name:     "Req",
		Method:   "GET",
		URL:      "https://example.com",
		BodyMode: "none",
		Headers:  []entity.KeyValue{{Key: "X-Trace", Value: "1"}},
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	// Attach a history row pointing at both root and request.
	hisRepo := NewHistoryRepository(client)
	if err := hisRepo.Save(ctx, &entity.HistoryItem{
		RootFolderID: &rootID,
		RequestID:    &reqID,
		Method:       "GET",
		URL:          "https://example.com",
		StatusCode:   200,
	}); err != nil {
		t.Fatalf("save history: %v", err)
	}

	if err := repo.DeleteByID(ctx, rootID); err != nil {
		t.Fatalf("delete root subtree: %v", err)
	}

	// Every folder in subtree is gone.
	roots, _ := repo.ListRoots(ctx)
	if len(roots) != 0 {
		t.Fatalf("expected no roots, got %d", len(roots))
	}
	n, err := client.Folder.Query().Count(ctx)
	if err != nil || n != 0 {
		t.Fatalf("expected 0 folders, got %d (err %v)", n, err)
	}
	// Request gone.
	if cnt, _ := client.Request.Query().Count(ctx); cnt != 0 {
		t.Fatalf("expected 0 requests, got %d", cnt)
	}
	// History row kept but FKs cleared.
	summaries, err := hisRepo.ListSummaries(ctx, nil)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("history should survive, got %d rows", len(summaries))
	}
	if summaries[0].RootFolderID != nil || summaries[0].RequestID != nil {
		t.Fatalf("FKs should be NULL after folder delete, got root=%v request=%v",
			summaries[0].RootFolderID, summaries[0].RequestID)
	}
}

func TestFolderRepo_ResolveRootID(t *testing.T) {
	ctx, _, repo := newFolderTestRig(t)
	rootID, _ := repo.Create(ctx, &entity.FolderItem{Name: "Root"})
	childID, _ := repo.Create(ctx, &entity.FolderItem{Name: "Child", ParentID: &rootID})
	grandID, _ := repo.Create(ctx, &entity.FolderItem{Name: "Grand", ParentID: &childID})

	got, err := repo.ResolveRootID(ctx, grandID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != rootID {
		t.Fatalf("root mismatch, want %s got %s", rootID, got)
	}
}

func TestFolderRepo_ListRootsOrderedBySortOrderThenName(t *testing.T) {
	ctx, client, repo := newFolderTestRig(t)
	_, _ = repo.Create(ctx, &entity.FolderItem{Name: "Bravo"})
	_, _ = repo.Create(ctx, &entity.FolderItem{Name: "Alpha"})
	_, _ = repo.Create(ctx, &entity.FolderItem{Name: "Charlie"})

	// Ent auto-assigns sort_order via nextSortOrderAppend (0,1,2), so default order = creation order.
	roots, err := repo.ListRoots(ctx)
	if err != nil {
		t.Fatalf("list roots: %v", err)
	}
	gotNames := []string{}
	for _, r := range roots {
		gotNames = append(gotNames, r.Name)
	}
	wantByInsertion := []string{"Bravo", "Alpha", "Charlie"}
	if !equalStrSlice(gotNames, wantByInsertion) {
		t.Fatalf("want %v (sort_order=insertion), got %v", wantByInsertion, gotNames)
	}

	// Force equal sort_order to check name tiebreak.
	if _, err := client.Folder.Update().SetSortOrder(0).Save(ctx); err != nil {
		t.Fatalf("normalize sort_order: %v", err)
	}
	roots, _ = repo.ListRoots(ctx)
	gotNames = gotNames[:0]
	for _, r := range roots {
		gotNames = append(gotNames, r.Name)
	}
	if !equalStrSlice(gotNames, []string{"Alpha", "Bravo", "Charlie"}) {
		t.Fatalf("name tiebreak failed, got %v", gotNames)
	}
}

func TestFolderRepo_MoveToParentAndReorder(t *testing.T) {
	ctx, _, repo := newFolderTestRig(t)

	r1, _ := repo.Create(ctx, &entity.FolderItem{Name: "R1"})
	r2, _ := repo.Create(ctx, &entity.FolderItem{Name: "R2"})
	childA, _ := repo.Create(ctx, &entity.FolderItem{Name: "A", ParentID: &r1})
	childB, _ := repo.Create(ctx, &entity.FolderItem{Name: "B", ParentID: &r1})
	childC, _ := repo.Create(ctx, &entity.FolderItem{Name: "C", ParentID: &r1})

	// Move childA under r2.
	if err := repo.MoveToParent(ctx, childA, strPtr(r2)); err != nil {
		t.Fatalf("move: %v", err)
	}
	r1Kids, _ := repo.ListChildren(ctx, r1)
	if len(r1Kids) != 2 {
		t.Fatalf("r1 should have 2 kids after move, got %d", len(r1Kids))
	}
	r2Kids, _ := repo.ListChildren(ctx, r2)
	if len(r2Kids) != 1 || r2Kids[0].ID != childA {
		t.Fatalf("r2 should now hold childA, got %+v", r2Kids)
	}

	// Reorder: put childC before childB under r1.
	if err := repo.ReorderFolderSibling(ctx, childC, r1, childB); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	r1Kids, _ = repo.ListChildren(ctx, r1)
	if len(r1Kids) != 2 || r1Kids[0].ID != childC || r1Kids[1].ID != childB {
		t.Fatalf("reorder failed: %+v", r1Kids)
	}

	// Move childA back to root (parent_id = nil).
	if err := repo.MoveToParent(ctx, childA, nil); err != nil {
		t.Fatalf("move to root: %v", err)
	}
	roots, _ := repo.ListRoots(ctx)
	seenA := false
	for _, f := range roots {
		if f.ID == childA {
			seenA = true
		}
	}
	if !seenA {
		t.Fatalf("childA should now be a root, got %+v", roots)
	}
}

func TestFolderRepo_ReorderAppendAtEnd(t *testing.T) {
	ctx, _, repo := newFolderTestRig(t)
	r, _ := repo.Create(ctx, &entity.FolderItem{Name: "R"})
	a, _ := repo.Create(ctx, &entity.FolderItem{Name: "A", ParentID: &r})
	b, _ := repo.Create(ctx, &entity.FolderItem{Name: "B", ParentID: &r})
	c, _ := repo.Create(ctx, &entity.FolderItem{Name: "C", ParentID: &r})

	// Empty insertBeforeID => append at end. Move A to the end.
	if err := repo.ReorderFolderSibling(ctx, a, r, ""); err != nil {
		t.Fatalf("reorder append: %v", err)
	}
	kids, _ := repo.ListChildren(ctx, r)
	if kids[0].ID != b || kids[1].ID != c || kids[2].ID != a {
		t.Fatalf("append-at-end failed: %+v", kids)
	}
}

func TestFolderRepo_SearchByNameTruncation(t *testing.T) {
	ctx, _, repo := newFolderTestRig(t)
	// Create 5 folders with "foo" in their names + 1 distractor.
	for _, n := range []string{"foo-1", "FooBar", "bar-foo", "blah", "Hello Foo"} {
		if _, err := repo.Create(ctx, &entity.FolderItem{Name: n}); err != nil {
			t.Fatalf("create %s: %v", n, err)
		}
	}

	rows, truncated, err := repo.SearchByName(ctx, "foo", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if truncated {
		t.Fatalf("should not truncate with limit 10, got %d rows", len(rows))
	}
	if len(rows) != 4 {
		t.Fatalf("expected 4 foo matches, got %d (%v)", len(rows), rowNames(rows))
	}

	// Limit smaller than matches → truncated = true.
	rows, truncated, err = repo.SearchByName(ctx, "foo", 2)
	if err != nil {
		t.Fatalf("search limited: %v", err)
	}
	if !truncated || len(rows) != 2 {
		t.Fatalf("expected truncated=true rows=2, got truncated=%v rows=%d", truncated, len(rows))
	}

	// Empty query → nil results, not an error.
	rows, truncated, err = repo.SearchByName(ctx, "   ", 5)
	if err != nil || rows != nil || truncated {
		t.Fatalf("empty query: rows=%v truncated=%v err=%v", rows, truncated, err)
	}
}

func rowNames(rows []*entity.FolderItem) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Name)
	}
	return out
}

func equalStrSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
