package repository

import (
	"context"
	"testing"

	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/testutil"
)

func newRuleRig(t *testing.T) (context.Context, FolderRepository, RequestRepository, RequestRuleRepository) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	return ctx, NewFolderRepository(client), NewRequestRepository(client), NewRequestRuleRepository(client)
}

func seedRuleRequest(t *testing.T, ctx context.Context, folders FolderRepository, reqs RequestRepository) string {
	t.Helper()
	fid, err := folders.Create(ctx, &entity.FolderItem{Name: "F"})
	if err != nil {
		t.Fatalf("folder: %v", err)
	}
	id, err := reqs.CreateFull(ctx, &entity.SavedRequestFull{
		FolderID: fid,
		Name:     "Req",
		Method:   "GET",
		URL:      "https://api.example.com/x",
	})
	if err != nil {
		t.Fatalf("create req: %v", err)
	}
	return id
}

func TestRuleRepo_CapturesReplaceAll(t *testing.T) {
	ctx, folders, reqs, rules := newRuleRig(t)
	rid := seedRuleRequest(t, ctx, folders, reqs)

	out, err := rules.SaveCaptures(ctx, rid, []entity.RequestCaptureInput{
		{Name: "token", Source: "json_body", Expression: "$.token", TargetScope: "environment", TargetVariable: "TOKEN", Enabled: true, SortOrder: 0},
		{Name: "id", Source: "header", Expression: "X-Id", TargetScope: "memory", TargetVariable: "X_ID", Enabled: true, SortOrder: 1},
		{Name: "", Source: "status", TargetVariable: "should_skip"}, // skipped (empty name)
	})
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 stored captures, got %d", len(out))
	}
	if out[0].Name != "token" || out[1].Name != "id" {
		t.Fatalf("sort order broken: %+v", out)
	}

	out, err = rules.SaveCaptures(ctx, rid, []entity.RequestCaptureInput{
		{Name: "only", Source: "json_body", Expression: "$.x", TargetScope: "environment", TargetVariable: "ONLY", Enabled: false, SortOrder: 0},
	})
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if len(out) != 1 || out[0].Name != "only" || out[0].Enabled {
		t.Fatalf("replace-all broken: %+v", out)
	}
}

func TestRuleRepo_AssertionsPersistOperatorAndExpected(t *testing.T) {
	ctx, folders, reqs, rules := newRuleRig(t)
	rid := seedRuleRequest(t, ctx, folders, reqs)

	out, err := rules.SaveAssertions(ctx, rid, []entity.RequestAssertionInput{
		{Name: "status ok", Source: "status", Operator: "eq", Expected: "200", Enabled: true, SortOrder: 0},
		{Name: "json has token", Source: "json_body", Expression: "$.token", Operator: "exists", Enabled: true, SortOrder: 1},
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 assertions, got %d", len(out))
	}
	if out[0].Operator != "eq" || out[0].Expected != "200" {
		t.Fatalf("first row corrupted: %+v", out[0])
	}
	if out[1].Source != "json_body" || out[1].Expression != "$.token" {
		t.Fatalf("second row corrupted: %+v", out[1])
	}
}

func TestRuleRepo_DeletingRequestCascadesRules(t *testing.T) {
	ctx, folders, reqs, rules := newRuleRig(t)
	rid := seedRuleRequest(t, ctx, folders, reqs)

	if _, err := rules.SaveCaptures(ctx, rid, []entity.RequestCaptureInput{
		{Name: "token", Source: "json_body", Expression: "$.t", TargetScope: "environment", TargetVariable: "T"},
	}); err != nil {
		t.Fatalf("seed cap: %v", err)
	}
	if _, err := rules.SaveAssertions(ctx, rid, []entity.RequestAssertionInput{
		{Name: "ok", Source: "status", Operator: "eq", Expected: "200"},
	}); err != nil {
		t.Fatalf("seed asrt: %v", err)
	}

	if err := reqs.DeleteByID(ctx, rid); err != nil {
		t.Fatalf("delete request: %v", err)
	}

	caps, err := rules.ListCaptures(ctx, rid)
	if err != nil {
		t.Fatalf("list cap: %v", err)
	}
	if len(caps) != 0 {
		t.Fatalf("captures should cascade, got %d", len(caps))
	}
	asrts, err := rules.ListAssertions(ctx, rid)
	if err != nil {
		t.Fatalf("list asrt: %v", err)
	}
	if len(asrts) != 0 {
		t.Fatalf("assertions should cascade, got %d", len(asrts))
	}
}
