package usecase

import (
	"context"
	"errors"
	"testing"

	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/testutil"
)

func newFolderUC(t *testing.T) (context.Context, FolderUsecase, repository.FolderRepository) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	folders := repository.NewFolderRepository(client)
	return ctx, NewFolderUsecase(folders), folders
}

func asAppErr(err error) *apperror.AppError {
	var ae *apperror.AppError
	if errors.As(err, &ae) {
		return ae
	}
	return nil
}

func TestFolderUC_CreateRejectsDuplicateRootName(t *testing.T) {
	ctx, uc, _ := newFolderUC(t)
	if _, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Root"}); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Root"})
	if err == nil {
		t.Fatal("expected duplicate root name error")
	}
	if ae := asAppErr(err); ae == nil || ae.Code != "FOL_301" {
		t.Fatalf("want FOL_301, got %+v", err)
	}
}

func TestFolderUC_CreateRejectsEmptyName(t *testing.T) {
	ctx, uc, _ := newFolderUC(t)
	if _, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "   "}); err == nil {
		t.Fatal("expected empty name error")
	}
}

func TestFolderUC_CreateChildRequiresExistingParent(t *testing.T) {
	ctx, uc, _ := newFolderUC(t)
	bogus := "11111111-1111-1111-1111-111111111111"
	_, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "X", ParentID: &bogus})
	if err == nil {
		t.Fatal("expected parent-not-found error")
	}
}

func TestFolderUC_CreateChildDuplicateUnderSameParent(t *testing.T) {
	ctx, uc, _ := newFolderUC(t)
	root, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Root"})
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	if _, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "C", ParentID: &root.ID}); err != nil {
		t.Fatalf("first child: %v", err)
	}
	_, err = uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "C", ParentID: &root.ID})
	if ae := asAppErr(err); ae == nil || ae.Code != "FOL_302" {
		t.Fatalf("want FOL_302, got %+v", err)
	}
}

func TestFolderUC_MoveCannotMoveIntoOwnSubtree(t *testing.T) {
	ctx, uc, _ := newFolderUC(t)
	root, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Root"})
	child, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Child", ParentID: &root.ID})
	grand, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Grand", ParentID: &child.ID})

	// Move root into grand → own subtree → must fail.
	if err := uc.MoveFolder(ctx, root.ID, grand.ID); err == nil {
		t.Fatal("expected error moving folder into its own subtree")
	}
	// Move into self → must fail.
	if err := uc.MoveFolder(ctx, root.ID, root.ID); err == nil {
		t.Fatal("expected error moving folder into itself")
	}
}

func TestFolderUC_MoveRejectsNameConflictInDestination(t *testing.T) {
	ctx, uc, _ := newFolderUC(t)
	rootA, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "A"})
	rootB, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "B"})
	// Both roots have a "Shared" child.
	if _, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Shared", ParentID: &rootA.ID}); err != nil {
		t.Fatalf("A.Shared: %v", err)
	}
	sharedB, err := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Shared", ParentID: &rootB.ID})
	if err != nil {
		t.Fatalf("B.Shared: %v", err)
	}

	// Moving B.Shared under A would collide with A.Shared.
	if err := uc.MoveFolder(ctx, sharedB.ID, rootA.ID); err == nil {
		t.Fatal("expected child name conflict on move")
	} else if ae := asAppErr(err); ae == nil || ae.Code != "FOL_302" {
		t.Fatalf("want FOL_302, got %+v", err)
	}
}

func TestFolderUC_ReorderFolder(t *testing.T) {
	ctx, uc, _ := newFolderUC(t)
	root, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Root"})
	a, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "A", ParentID: &root.ID})
	b, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "B", ParentID: &root.ID})
	c, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "C", ParentID: &root.ID})

	// Move C before A: expected order C, A, B.
	if err := uc.ReorderFolder(ctx, c.ID, root.ID, a.ID); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	kids, _ := uc.ListChildFolders(ctx, root.ID)
	if len(kids) != 3 || kids[0].ID != c.ID || kids[1].ID != a.ID || kids[2].ID != b.ID {
		t.Fatalf("reorder wrong, got %+v", kids)
	}
}

func TestFolderUC_DeleteFolderRemovesSubtree(t *testing.T) {
	ctx, uc, folders := newFolderUC(t)
	root, _ := uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Root"})
	_, _ = uc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Child", ParentID: &root.ID})

	if err := uc.DeleteFolder(ctx, root.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ := folders.ListRoots(ctx)
	if len(list) != 0 {
		t.Fatalf("roots should be empty, got %+v", list)
	}
}

func TestFolderUC_DuplicateFolderCopiesTreeAndRequests(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	folders := repository.NewFolderRepository(client)
	reqs := repository.NewRequestRepository(client)
	fuc := NewFolderUsecaseWithRequests(folders, reqs)
	ruc := NewRequestUsecase(folders, reqs)

	root, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Root"})
	child, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "Child", ParentID: &root.ID})
	if _, err := ruc.CreateRequest(ctx, &entity.SavedRequestFull{
		FolderID: child.ID,
		Name:     "Req",
		Method:   "GET",
		URL:      "https://e.test",
		BodyMode: "none",
	}); err != nil {
		t.Fatalf("create request: %v", err)
	}

	dupRoot, err := fuc.DuplicateFolder(ctx, root.ID)
	if err != nil {
		t.Fatalf("duplicate folder: %v", err)
	}
	if dupRoot.ID == root.ID || dupRoot.Name != "Root (copy)" {
		t.Fatalf("unexpected duplicate root: %+v", dupRoot)
	}
	dupChildren, err := fuc.ListChildFolders(ctx, dupRoot.ID)
	if err != nil {
		t.Fatalf("list dup children: %v", err)
	}
	if len(dupChildren) != 1 || dupChildren[0].Name != "Child" {
		t.Fatalf("child tree not copied: %+v", dupChildren)
	}
	dupReqs, err := ruc.ListRequestsInFolder(ctx, dupChildren[0].ID)
	if err != nil {
		t.Fatalf("list dup requests: %v", err)
	}
	if len(dupReqs) != 1 || dupReqs[0].Name != "Req" {
		t.Fatalf("request not copied: %+v", dupReqs)
	}
}
