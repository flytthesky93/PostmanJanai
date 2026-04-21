package usecase

import (
	"context"
	"testing"

	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/testutil"
)

func newEnvUC(t *testing.T) (context.Context, EnvironmentUsecase, repository.EnvironmentRepository) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	repo := repository.NewEnvironmentRepository(client)
	return ctx, NewEnvironmentUsecase(repo), repo
}

func TestEnvUC_CreateRejectsDuplicateName(t *testing.T) {
	ctx, uc, _ := newEnvUC(t)
	if _, err := uc.Create(ctx, "dev", ""); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := uc.Create(ctx, "dev", "")
	if ae := asAppErr(err); ae == nil || ae.Code != "ENV_602" {
		t.Fatalf("want ENV_602, got %+v", err)
	}
}

func TestEnvUC_CreateRejectsEmptyName(t *testing.T) {
	ctx, uc, _ := newEnvUC(t)
	_, err := uc.Create(ctx, "   ", "")
	if err == nil {
		t.Fatal("expected empty name error")
	}
}

func TestEnvUC_SaveVariablesRejectsDuplicateKey(t *testing.T) {
	ctx, uc, _ := newEnvUC(t)
	env, _ := uc.Create(ctx, "env", "")
	err := uc.SaveVariables(ctx, env.ID, []entity.EnvVariableInput{
		{Key: "k", Value: "1", Enabled: true},
		{Key: "K", Value: "2", Enabled: true},
	})
	if ae := asAppErr(err); ae == nil || ae.Code != "ENV_603" {
		t.Fatalf("want ENV_603 (case-insensitive duplicate), got %+v", err)
	}
}

func TestEnvUC_SetActiveEnforcesSingleActive(t *testing.T) {
	ctx, uc, repo := newEnvUC(t)
	a, _ := uc.Create(ctx, "A", "")
	b, _ := uc.Create(ctx, "B", "")

	if err := uc.SetActive(ctx, a.ID); err != nil {
		t.Fatalf("set A: %v", err)
	}
	if err := uc.SetActive(ctx, b.ID); err != nil {
		t.Fatalf("set B: %v", err)
	}

	all, _ := repo.ListSummaries(ctx)
	active := 0
	for _, s := range all {
		if s.IsActive {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("exactly one env must be active, got %d", active)
	}
	sum, _ := uc.GetActiveSummary(ctx)
	if sum == nil || sum.ID != b.ID {
		t.Fatalf("active should be B, got %+v", sum)
	}
}

func TestEnvUC_DeleteReturnsNotFoundForUnknownID(t *testing.T) {
	ctx, uc, _ := newEnvUC(t)
	err := uc.Delete(ctx, "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected not-found")
	}
	if ae := asAppErr(err); ae == nil || ae.Code != "ENV_601" {
		t.Fatalf("want ENV_601, got %+v", err)
	}
}
