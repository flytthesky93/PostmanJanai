package usecase

import (
	"context"
	"testing"

	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/testutil"
)

func newRequestUCRig(t *testing.T) (context.Context, RequestUsecase, FolderUsecase, repository.FolderRepository, repository.RequestRepository) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	folders := repository.NewFolderRepository(client)
	reqs := repository.NewRequestRepository(client)
	return ctx, NewRequestUsecase(folders, reqs), NewFolderUsecase(folders), folders, reqs
}

func TestRequestUC_CreateRejectsInvalidFolder(t *testing.T) {
	ctx, uc, _, _, _ := newRequestUCRig(t)
	_, err := uc.CreateRequest(ctx, &entity.SavedRequestFull{
		FolderID: "not-a-uuid",
		Name:     "X",
		Method:   "GET",
		URL:      "https://e.com",
	})
	if err == nil {
		t.Fatal("expected invalid folder error")
	}
}

func TestRequestUC_CreateRejectsDuplicateNameInSameFolder(t *testing.T) {
	ctx, uc, fuc, _, _ := newRequestUCRig(t)
	root, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "R"})

	if _, err := uc.CreateRequest(ctx, &entity.SavedRequestFull{
		FolderID: root.ID, Name: "Req", Method: "GET", URL: "https://e.com",
	}); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := uc.CreateRequest(ctx, &entity.SavedRequestFull{
		FolderID: root.ID, Name: "Req", Method: "GET", URL: "https://e.com",
	})
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if ae := asAppErr(err); ae == nil || ae.Code != "REQ_502" {
		t.Fatalf("want REQ_502, got %+v", err)
	}
}

func TestRequestUC_CreateNormalizesEmptyFields(t *testing.T) {
	ctx, uc, fuc, _, _ := newRequestUCRig(t)
	root, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "R"})
	got, err := uc.CreateRequest(ctx, &entity.SavedRequestFull{
		FolderID: root.ID,
		Name:     "  R  ",
		Method:   "   ",
		URL:      "   ",
		BodyMode: "  ",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.Method != "GET" {
		t.Fatalf("want method GET, got %q", got.Method)
	}
	if got.URL != "https://" {
		t.Fatalf("want URL https://, got %q", got.URL)
	}
	if got.BodyMode != "none" {
		t.Fatalf("want BodyMode none, got %q", got.BodyMode)
	}
	if got.Name != "R" {
		t.Fatalf("want trimmed name R, got %q", got.Name)
	}
}

func TestRequestUC_UpdateRejectsNameConflict(t *testing.T) {
	ctx, uc, fuc, _, _ := newRequestUCRig(t)
	root, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "R"})
	_, err := uc.CreateRequest(ctx, &entity.SavedRequestFull{FolderID: root.ID, Name: "A", Method: "GET", URL: "https://e"})
	if err != nil {
		t.Fatalf("A: %v", err)
	}
	reqB, err := uc.CreateRequest(ctx, &entity.SavedRequestFull{FolderID: root.ID, Name: "B", Method: "GET", URL: "https://e"})
	if err != nil {
		t.Fatalf("B: %v", err)
	}
	err = uc.UpdateRequest(ctx, &entity.SavedRequestFull{
		ID: reqB.ID, FolderID: root.ID, Name: "A", Method: "GET", URL: "https://e",
	})
	if err == nil {
		t.Fatal("expected name-conflict error")
	}
	if ae := asAppErr(err); ae == nil || ae.Code != "REQ_502" {
		t.Fatalf("want REQ_502, got %+v", err)
	}
}

func TestRequestUC_MoveRequestDestinationNameCollision(t *testing.T) {
	ctx, uc, fuc, _, _ := newRequestUCRig(t)
	f1, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "F1"})
	f2, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "F2"})
	r1, _ := uc.CreateRequest(ctx, &entity.SavedRequestFull{FolderID: f1.ID, Name: "Req", Method: "GET", URL: "https://e"})
	_, _ = uc.CreateRequest(ctx, &entity.SavedRequestFull{FolderID: f2.ID, Name: "Req", Method: "GET", URL: "https://e"})

	if err := uc.MoveRequest(ctx, r1.ID, f2.ID); err == nil {
		t.Fatal("expected move to fail on name collision")
	}
}

func TestRequestUC_MoveRequestNoopToSameFolder(t *testing.T) {
	ctx, uc, fuc, _, _ := newRequestUCRig(t)
	f, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "F"})
	r, _ := uc.CreateRequest(ctx, &entity.SavedRequestFull{FolderID: f.ID, Name: "Req", Method: "GET", URL: "https://e"})
	if err := uc.MoveRequest(ctx, r.ID, f.ID); err != nil {
		t.Fatalf("same-folder move should be no-op, got %v", err)
	}
}

func TestRequestUC_DeleteRequest(t *testing.T) {
	ctx, uc, fuc, _, reqs := newRequestUCRig(t)
	f, _ := fuc.CreateFolder(ctx, &entity.CreateFolderInput{Name: "F"})
	r, _ := uc.CreateRequest(ctx, &entity.SavedRequestFull{FolderID: f.ID, Name: "R", Method: "GET", URL: "https://e"})

	if err := uc.DeleteRequest(ctx, r.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ := reqs.ListByFolder(ctx, f.ID)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}
