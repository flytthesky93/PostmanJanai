package repository

import (
	"context"
	"testing"

	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/testutil"
)

func newEnvRig(t *testing.T) (context.Context, EnvironmentRepository) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	return ctx, NewEnvironmentRepository(client)
}

func TestEnvRepo_CreateAndNameTaken(t *testing.T) {
	ctx, repo := newEnvRig(t)
	full, err := repo.Create(ctx, "dev", "dev env")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if full.Name != "dev" || full.IsActive {
		t.Fatalf("unexpected created env: %+v", full.EnvironmentSummary)
	}

	taken, err := repo.NameTaken(ctx, "dev", nil)
	if err != nil || !taken {
		t.Fatalf("NameTaken want true err nil, got %v err %v", taken, err)
	}
	taken, _ = repo.NameTaken(ctx, "dev", &full.ID)
	if taken {
		t.Fatal("NameTaken excluding self should be false")
	}
}

func TestEnvRepo_SaveVariablesReplaces(t *testing.T) {
	ctx, repo := newEnvRig(t)
	full, _ := repo.Create(ctx, "env", "")

	err := repo.SaveVariables(ctx, full.ID, []entity.EnvVariableInput{
		{Key: "base_url", Value: "https://a", Enabled: true},
		{Key: "token", Value: "t1", Enabled: true},
	})
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	got, err := repo.GetFull(ctx, full.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Variables) != 2 {
		t.Fatalf("want 2 vars, got %d", len(got.Variables))
	}

	// Second save fully replaces.
	err = repo.SaveVariables(ctx, full.ID, []entity.EnvVariableInput{
		{Key: "base_url", Value: "https://b", Enabled: false},
	})
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	got, _ = repo.GetFull(ctx, full.ID)
	if len(got.Variables) != 1 {
		t.Fatalf("replace failed, got %+v", got.Variables)
	}
	if got.Variables[0].Key != "base_url" || got.Variables[0].Value != "https://b" || got.Variables[0].Enabled {
		t.Fatalf("variable row wrong: %+v", got.Variables[0])
	}
}

func TestEnvRepo_SetActiveAndActiveVariableMap(t *testing.T) {
	ctx, repo := newEnvRig(t)
	envA, _ := repo.Create(ctx, "A", "")
	envB, _ := repo.Create(ctx, "B", "")

	_ = repo.SaveVariables(ctx, envA.ID, []entity.EnvVariableInput{
		{Key: "only_a", Value: "va", Enabled: true},
		{Key: "disabled_key", Value: "nope", Enabled: false},
	})
	_ = repo.SaveVariables(ctx, envB.ID, []entity.EnvVariableInput{
		{Key: "only_b", Value: "vb", Enabled: true},
	})

	// No env active → empty map.
	m, err := repo.ActiveVariableMap(ctx)
	if err != nil {
		t.Fatalf("active var map empty: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("want empty map, got %v", m)
	}

	// Activate A.
	if err := repo.SetActive(ctx, envA.ID); err != nil {
		t.Fatalf("set active A: %v", err)
	}
	sum, err := repo.GetActiveSummary(ctx)
	if err != nil || sum == nil || sum.ID != envA.ID {
		t.Fatalf("active summary mismatch: %+v err=%v", sum, err)
	}
	m, _ = repo.ActiveVariableMap(ctx)
	if m["only_a"] != "va" {
		t.Fatalf("only_a not in map: %v", m)
	}
	if _, ok := m["disabled_key"]; ok {
		t.Fatal("disabled variable should not appear in active map")
	}

	// Switch to B — A must auto-deactivate.
	if err := repo.SetActive(ctx, envB.ID); err != nil {
		t.Fatalf("set active B: %v", err)
	}
	all, _ := repo.ListSummaries(ctx)
	activeCount := 0
	for _, s := range all {
		if s.IsActive {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("exactly one env must be active, got %d", activeCount)
	}
	m, _ = repo.ActiveVariableMap(ctx)
	if m["only_b"] != "vb" || m["only_a"] != "" {
		t.Fatalf("active map after switch wrong: %v", m)
	}

	// Clear active.
	if err := repo.ClearActive(ctx); err != nil {
		t.Fatalf("clear active: %v", err)
	}
	sum, _ = repo.GetActiveSummary(ctx)
	if sum != nil {
		t.Fatalf("summary should be nil after clear, got %+v", sum)
	}
}

func TestEnvRepo_DeleteRemovesVariables(t *testing.T) {
	ctx, repo := newEnvRig(t)
	full, _ := repo.Create(ctx, "E", "")
	_ = repo.SaveVariables(ctx, full.ID, []entity.EnvVariableInput{
		{Key: "x", Value: "y", Enabled: true},
	})

	if err := repo.Delete(ctx, full.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ := repo.GetFull(ctx, full.ID)
	if got != nil {
		t.Fatalf("env should be gone, got %+v", got)
	}
}
