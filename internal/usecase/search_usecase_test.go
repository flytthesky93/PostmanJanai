package usecase

import (
	"PostmanJanai/internal/entity"
	"reflect"
	"testing"
)

func ptrStr(s string) *string { return &s }

func TestPathForFolder(t *testing.T) {
	// root (A) ─┬─ B ─── C
	//           └─ D
	index := buildFolderIndex([]*entity.FolderItem{
		{ID: "A", Name: "Root"},
		{ID: "B", Name: "Sub1", ParentID: ptrStr("A")},
		{ID: "C", Name: "Leaf", ParentID: ptrStr("B")},
		{ID: "D", Name: "Sub2", ParentID: ptrStr("A")},
	})

	cases := []struct {
		name     string
		id       string
		wantPath []string
		wantRoot string
	}{
		{"root", "A", []string{"Root"}, "A"},
		{"mid", "B", []string{"Root", "Sub1"}, "A"},
		{"leaf", "C", []string{"Root", "Sub1", "Leaf"}, "A"},
		{"sibling", "D", []string{"Root", "Sub2"}, "A"},
		{"missing", "Z", nil, ""},
		{"empty", "", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, root := pathForFolder(index, c.id)
			if !reflect.DeepEqual(path, c.wantPath) {
				t.Fatalf("path = %v, want %v", path, c.wantPath)
			}
			if root != c.wantRoot {
				t.Fatalf("root = %q, want %q", root, c.wantRoot)
			}
		})
	}
}

func TestPathForFolder_CycleSafe(t *testing.T) {
	// Pathological cycle A → B → A; helper must terminate without panicking.
	index := buildFolderIndex([]*entity.FolderItem{
		{ID: "A", Name: "A", ParentID: ptrStr("B")},
		{ID: "B", Name: "B", ParentID: ptrStr("A")},
	})
	path, root := pathForFolder(index, "A")
	if len(path) == 0 {
		t.Fatal("expected non-empty path")
	}
	if root == "" {
		t.Fatal("expected non-empty root")
	}
}
