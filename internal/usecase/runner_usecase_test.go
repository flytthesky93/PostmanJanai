package usecase

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/testutil"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

// stubExecutor returns deterministic responses by URL pattern. Tests use
// substring keys so they survive env-var substitution at runtime.
type stubExecutor struct {
	mu        sync.Mutex
	calls     []*entity.HTTPExecuteInput
	responses map[string]*entity.HTTPExecuteResult
}

func newStubExecutor() *stubExecutor {
	return &stubExecutor{responses: map[string]*entity.HTTPExecuteResult{}}
}

func (s *stubExecutor) on(urlSubstring string, res *entity.HTTPExecuteResult) {
	s.responses[urlSubstring] = res
}

func (s *stubExecutor) Execute(ctx context.Context, in *entity.HTTPExecuteInput) (*entity.HTTPExecuteResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, in)
	for sub, res := range s.responses {
		if strings.Contains(in.URL, sub) {
			out := *res
			out.FinalURL = in.URL
			return &out, nil
		}
	}
	return nil, errors.New("no stub registered for " + in.URL)
}

type captureEmitter struct {
	mu     sync.Mutex
	events []struct {
		name    string
		payload any
	}
}

func (c *captureEmitter) Emit(name string, payload any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, struct {
		name    string
		payload any
	}{name, payload})
}

func newRunnerUC(t *testing.T, exec RunnerHTTPExecutor) (
	context.Context,
	RunnerUsecase,
	repository.FolderRepository,
	repository.RequestRepository,
	repository.RequestRuleRepository,
	repository.EnvironmentRepository,
	repository.RunnerRepository,
) {
	t.Helper()
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	folders := repository.NewFolderRepository(client)
	requests := repository.NewRequestRepository(client)
	rules := repository.NewRequestRuleRepository(client)
	envRepo := repository.NewEnvironmentRepository(client, nil)
	runs := repository.NewRunnerRepository(client)
	uc := NewRunnerUsecase(folders, requests, rules, envRepo, runs, exec)
	return ctx, uc, folders, requests, rules, envRepo, runs
}

func mustCreateRoot(t *testing.T, folders repository.FolderRepository, name string) *entity.FolderItem {
	t.Helper()
	id, err := folders.Create(context.Background(), &entity.FolderItem{Name: name})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	got, err := folders.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("get root: %v", err)
	}
	return got
}

func mustCreateRequest(t *testing.T, requests repository.RequestRepository, folderID, name, url string) string {
	t.Helper()
	id, err := requests.CreateFull(context.Background(), &entity.SavedRequestFull{
		FolderID: folderID,
		Name:     name,
		Method:   "GET",
		URL:      url,
		BodyMode: "none",
	})
	if err != nil {
		t.Fatalf("create request %s: %v", name, err)
	}
	return id
}

func TestRunner_RunFolder_AssertionsAndCaptureEnvScope(t *testing.T) {
	exec := newStubExecutor()
	ctx, uc, folders, requests, rules, envRepo, _ := newRunnerUC(t, exec)
	root := mustCreateRoot(t, folders, "Run me")

	loginID := mustCreateRequest(t, requests, root.ID, "login", "https://api.example.com/login")
	profileID := mustCreateRequest(t, requests, root.ID, "profile", "https://api.example.com/profile")

	// login -> capture token + assert 200
	if _, err := rules.SaveCaptures(ctx, loginID, []entity.RequestCaptureInput{
		{Name: "token", Source: constant.CaptureSourceJSONBody, Expression: "$.token", TargetScope: constant.CaptureScopeEnvironment, TargetVariable: "auth_token", Enabled: true},
	}); err != nil {
		t.Fatalf("save captures: %v", err)
	}
	if _, err := rules.SaveAssertions(ctx, loginID, []entity.RequestAssertionInput{
		{Name: "status 200", Source: constant.AssertionSourceStatus, Operator: constant.AssertionOpEq, Expected: "200", Enabled: true},
	}); err != nil {
		t.Fatalf("save assertions login: %v", err)
	}
	// profile -> assert 200 only (the URL itself contains no var, but capture must persist)
	if _, err := rules.SaveAssertions(ctx, profileID, []entity.RequestAssertionInput{
		{Name: "status 200", Source: constant.AssertionSourceStatus, Operator: constant.AssertionOpEq, Expected: "200", Enabled: true},
		{Name: "name eq", Source: constant.AssertionSourceJSONBody, Expression: "$.user.name", Operator: constant.AssertionOpEq, Expected: "Alice", Enabled: true},
	}); err != nil {
		t.Fatalf("save assertions profile: %v", err)
	}

	// Active environment for capture target.
	envFull, err := envRepo.Create(ctx, "default", "")
	if err != nil {
		t.Fatalf("create env: %v", err)
	}
	if err := envRepo.SetActive(ctx, envFull.ID); err != nil {
		t.Fatalf("set active: %v", err)
	}

	exec.on("/login", &entity.HTTPExecuteResult{
		StatusCode:        200,
		DurationMs:        12,
		ResponseSizeBytes: 32,
		ResponseBody:      `{"token":"tok-1"}`,
		ResponseHeaders:   []entity.KeyValue{{Key: "Content-Type", Value: "application/json"}},
	})
	exec.on("/profile", &entity.HTTPExecuteResult{
		StatusCode:        200,
		DurationMs:        7,
		ResponseSizeBytes: 64,
		ResponseBody:      `{"user":{"name":"Alice"}}`,
		ResponseHeaders:   []entity.KeyValue{{Key: "Content-Type", Value: "application/json"}},
	})

	emitter := &captureEmitter{}
	detail, err := uc.RunFolder(ctx, &entity.RunFolderInput{FolderID: root.ID}, emitter)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if detail.Status != constant.RunnerStatusCompleted {
		t.Fatalf("status = %s", detail.Status)
	}
	if detail.PassedCount != 2 {
		t.Errorf("passed = %d, want 2", detail.PassedCount)
	}
	if detail.FailedCount != 0 || detail.ErrorCount != 0 {
		t.Errorf("non-zero failure counters: %+v", detail)
	}
	if len(detail.Requests) != 2 {
		t.Fatalf("expected 2 result rows, got %d", len(detail.Requests))
	}
	// Capture must have persisted to the active environment.
	envBag, err := envRepo.ActiveVariableMap(ctx)
	if err != nil {
		t.Fatalf("env map: %v", err)
	}
	if envBag["auth_token"] != "tok-1" {
		t.Errorf("auth_token = %q want tok-1; bag=%v", envBag["auth_token"], envBag)
	}
	// At least 1 started + N request + finished events.
	gotEventNames := map[string]int{}
	for _, e := range emitter.events {
		gotEventNames[e.name]++
	}
	if gotEventNames[constant.RunnerEventStarted] < 1 || gotEventNames[constant.RunnerEventFinished] < 1 {
		t.Errorf("missing lifecycle events: %v", gotEventNames)
	}
	if gotEventNames[constant.RunnerEventRequestDone] < 2 {
		t.Errorf("expected at least 2 request events, got %d", gotEventNames[constant.RunnerEventRequestDone])
	}
}

func TestRunner_RunFolder_StopOnFail(t *testing.T) {
	exec := newStubExecutor()
	ctx, uc, folders, requests, rules, _, _ := newRunnerUC(t, exec)
	root := mustCreateRoot(t, folders, "stop")
	aID := mustCreateRequest(t, requests, root.ID, "a", "https://api.example.com/a")
	mustCreateRequest(t, requests, root.ID, "b", "https://api.example.com/b")

	if _, err := rules.SaveAssertions(ctx, aID, []entity.RequestAssertionInput{
		{Name: "status 200", Source: constant.AssertionSourceStatus, Operator: constant.AssertionOpEq, Expected: "200", Enabled: true},
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	exec.on("/a", &entity.HTTPExecuteResult{StatusCode: 500, ResponseBody: ""})
	exec.on("/b", &entity.HTTPExecuteResult{StatusCode: 200, ResponseBody: ""})

	detail, err := uc.RunFolder(ctx, &entity.RunFolderInput{FolderID: root.ID, StopOnFail: true}, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if detail.Status != constant.RunnerStatusFailed {
		t.Errorf("status = %s, want failed", detail.Status)
	}
	if len(detail.Requests) != 1 {
		t.Errorf("expected 1 request executed before stop, got %d", len(detail.Requests))
	}
}

func TestRunner_RunFolder_EmptyFolderRejected(t *testing.T) {
	exec := newStubExecutor()
	ctx, uc, folders, _, _, _, _ := newRunnerUC(t, exec)
	root := mustCreateRoot(t, folders, "empty")
	_, err := uc.RunFolder(ctx, &entity.RunFolderInput{FolderID: root.ID}, nil)
	if err == nil {
		t.Fatal("expected error for empty folder")
	}
}

func TestRunner_RunFolder_RecentList(t *testing.T) {
	exec := newStubExecutor()
	ctx, uc, folders, requests, _, _, runs := newRunnerUC(t, exec)
	root := mustCreateRoot(t, folders, "List me")
	mustCreateRequest(t, requests, root.ID, "a", "https://api.example.com/a")
	exec.on("/a", &entity.HTTPExecuteResult{StatusCode: 200, ResponseBody: ""})

	if _, err := uc.RunFolder(ctx, &entity.RunFolderInput{FolderID: root.ID}, nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	list, err := runs.ListRecent(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 recent run, got %d", len(list))
	}
}
