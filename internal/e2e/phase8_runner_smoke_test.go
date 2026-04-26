package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/service"
	"PostmanJanai/internal/testutil"
	"PostmanJanai/internal/usecase"
)

// captureEmitter records every payload the runner pushes so we can assert the
// frontend would receive started → request → finished in order.
type captureEmitter struct {
	mu     sync.Mutex
	events []emitEvent
}

type emitEvent struct {
	Name    string
	Payload any
}

func (c *captureEmitter) Emit(name string, payload any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, emitEvent{Name: name, Payload: payload})
}

// TestSmoke_Phase8_CaptureChainAndRunner is the Phase 8 sibling of TestSmoke_FullUserJourney.
// It proves three things end-to-end against real DB + httptest.Server:
//
//  1. A capture rule (json_body $.token → environment.TOKEN) extracts a value AND
//     persists it on the active environment so the very next request sees it.
//  2. An assertion rule (status eq 200) is recorded on the request row and used
//     to set passed/failed on the runner row.
//  3. The runner emits started + per-request + finished events and the resulting
//     run detail can be marshalled to both JSON and Markdown reports.
func TestSmoke_Phase8_CaptureChainAndRunner(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)

	folders := repository.NewFolderRepository(client)
	reqs := repository.NewRequestRepository(client)
	cipher, err := service.NewSecretCipher()
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	envs := repository.NewEnvironmentRepository(client, cipher)
	rules := repository.NewRequestRuleRepository(client)
	runs := repository.NewRunnerRepository(client)

	// --- Fake API: /login returns a token; /me echoes the bearer token.
	var loginHits, meHits int32
	var meAuth atomic.Value // string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			atomic.AddInt32(&loginHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"tkn-42","user":"alice"}`))
		case "/me":
			atomic.AddInt32(&meHits, 1)
			meAuth.Store(r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":7,"email":"alice@example.com"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// --- Folder + active env (seeded base_url so URLs in saved requests stay portable).
	rootID, fErr := folders.Create(ctx, &entity.FolderItem{Name: "Auth flow"})
	if fErr != nil {
		t.Fatalf("create folder: %v", fErr)
	}
	envFull, err := envs.Create(ctx, "local", "")
	if err != nil {
		t.Fatalf("create env: %v", err)
	}
	if err := envs.SaveVariables(ctx, envFull.ID, []entity.EnvVariableInput{
		{Key: "base_url", Value: srv.URL, Enabled: true},
	}); err != nil {
		t.Fatalf("save vars: %v", err)
	}
	if err := envs.SetActive(ctx, envFull.ID); err != nil {
		t.Fatalf("activate env: %v", err)
	}

	// --- 2 saved requests: 1) login that captures token, 2) /me using {{TOKEN}} via bearer.
	loginID, err := reqs.CreateFull(ctx, &entity.SavedRequestFull{
		FolderID: rootID,
		Name:     "01 Login",
		Method:   "POST",
		URL:      "{{base_url}}/login",
		BodyMode: string(entity.BodyModeRaw),
		RawBody:  ptr(`{"u":"alice"}`),
		Headers:  []entity.KeyValue{{Key: "Content-Type", Value: "application/json"}},
	})
	if err != nil {
		t.Fatalf("create login: %v", err)
	}
	meID, err := reqs.CreateFull(ctx, &entity.SavedRequestFull{
		FolderID: rootID,
		Name:     "02 Me",
		Method:   "GET",
		URL:      "{{base_url}}/me",
		Auth:     &entity.RequestAuth{Type: "bearer", BearerToken: "{{TOKEN}}"},
	})
	if err != nil {
		t.Fatalf("create me: %v", err)
	}

	// --- Rules: login captures $.token into environment.TOKEN, both assert status=200.
	if _, err := rules.SaveCaptures(ctx, loginID, []entity.RequestCaptureInput{
		{Name: "token", Source: constant.CaptureSourceJSONBody, Expression: "$.token", TargetScope: constant.CaptureScopeEnvironment, TargetVariable: "TOKEN", Enabled: true, SortOrder: 0},
	}); err != nil {
		t.Fatalf("save login captures: %v", err)
	}
	if _, err := rules.SaveAssertions(ctx, loginID, []entity.RequestAssertionInput{
		{Name: "login 200", Source: constant.AssertionSourceStatus, Operator: constant.AssertionOpEq, Expected: "200", Enabled: true, SortOrder: 0},
	}); err != nil {
		t.Fatalf("save login assertions: %v", err)
	}
	if _, err := rules.SaveAssertions(ctx, meID, []entity.RequestAssertionInput{
		{Name: "me 200", Source: constant.AssertionSourceStatus, Operator: constant.AssertionOpEq, Expected: "200", Enabled: true, SortOrder: 0},
		{Name: "me has email", Source: constant.AssertionSourceJSONBody, Expression: "$.email", Operator: constant.AssertionOpContains, Expected: "alice", Enabled: true, SortOrder: 1},
	}); err != nil {
		t.Fatalf("save me assertions: %v", err)
	}

	// --- Runner usecase wiring (real HTTPExecutor with no transport overrides).
	exec := service.NewHTTPExecutor(nil)
	uc := usecase.NewRunnerUsecase(folders, reqs, rules, envs, runs, exec)

	emitter := &captureEmitter{}
	detail, err := uc.RunFolder(ctx, &entity.RunFolderInput{
		FolderID:      rootID,
		EnvironmentID: envFull.ID,
		Notes:         "phase 8 smoke",
	}, emitter)
	if err != nil {
		t.Fatalf("RunFolder: %v", err)
	}

	if detail == nil || detail.TotalCount != 2 {
		t.Fatalf("want 2 requests in detail, got %+v", detail)
	}
	if detail.PassedCount != 2 || detail.FailedCount != 0 || detail.ErrorCount != 0 {
		t.Fatalf("totals mismatch: passed=%d failed=%d errored=%d", detail.PassedCount, detail.FailedCount, detail.ErrorCount)
	}
	if detail.Notes != "phase 8 smoke" {
		t.Fatalf("notes lost: %q", detail.Notes)
	}

	// Server should have been hit exactly once per request.
	if got := atomic.LoadInt32(&loginHits); got != 1 {
		t.Fatalf("login hits want 1 got %d", got)
	}
	if got := atomic.LoadInt32(&meHits); got != 1 {
		t.Fatalf("me hits want 1 got %d", got)
	}

	// The captured TOKEN must have flowed into the next request's Authorization header.
	auth, _ := meAuth.Load().(string)
	if auth != "Bearer tkn-42" {
		t.Fatalf("captured token did not chain into /me: %q", auth)
	}

	// And it must have persisted on the active environment for future runs.
	if vars, err := envs.ActiveVariableMap(ctx); err != nil {
		t.Fatalf("active vars: %v", err)
	} else if vars["TOKEN"] != "tkn-42" {
		t.Fatalf("env TOKEN not persisted, got %q", vars["TOKEN"])
	}

	// Per-request assertions are present and all passed.
	for _, r := range detail.Requests {
		if len(r.Assertions) == 0 {
			t.Fatalf("request %q missing assertions", r.RequestName)
		}
		for _, a := range r.Assertions {
			if !a.Passed {
				t.Fatalf("assertion failed unexpectedly on %q: %+v", r.RequestName, a)
			}
		}
	}
	// Login row carries the capture result.
	loginRow := findRow(detail.Requests, "01 Login")
	if loginRow == nil || len(loginRow.Captures) != 1 || loginRow.Captures[0].Value != "tkn-42" {
		t.Fatalf("login capture missing or wrong: %+v", loginRow)
	}

	// Phase 8.1 — request/response snapshots are persisted with the run so the
	// detail modal can show the raw payload without re-running the request.
	// Login: posts JSON body, server returns JSON with token.
	if !strings.Contains(loginRow.RequestBody, `"u":"alice"`) {
		t.Fatalf("login row should persist the resolved request body, got %q", loginRow.RequestBody)
	}
	if !strings.Contains(loginRow.ResponseBody, `"token":"tkn-42"`) {
		t.Fatalf("login row should persist the response body, got %q", loginRow.ResponseBody)
	}
	if !strings.Contains(loginRow.RequestHeadersJSON, "Content-Type") {
		t.Fatalf("login row should persist request headers, got %q", loginRow.RequestHeadersJSON)
	}
	if !strings.Contains(loginRow.ResponseHeadersJSON, "Content-Type") {
		t.Fatalf("login row should persist response headers, got %q", loginRow.ResponseHeadersJSON)
	}
	// /me uses bearer auth derived from the captured TOKEN — the resolved
	// request snapshot must contain the substituted bearer token, not the
	// `{{TOKEN}}` placeholder.
	meRow := findRow(detail.Requests, "02 Me")
	if meRow == nil {
		t.Fatalf("me row missing")
	}
	if !strings.Contains(meRow.RequestHeadersJSON, "Bearer tkn-42") {
		t.Fatalf("me row should persist resolved Authorization header, got %q", meRow.RequestHeadersJSON)
	}
	if strings.Contains(meRow.RequestHeadersJSON, "{{TOKEN}}") {
		t.Fatalf("me row leaked the unresolved {{TOKEN}} placeholder: %q", meRow.RequestHeadersJSON)
	}
	if !strings.Contains(meRow.ResponseBody, "alice@example.com") {
		t.Fatalf("me row should persist response body, got %q", meRow.ResponseBody)
	}

	// Wails event order: started → request × N → finished.
	emitter.mu.Lock()
	events := append([]emitEvent(nil), emitter.events...)
	emitter.mu.Unlock()
	if len(events) < 4 {
		t.Fatalf("want >=4 events (started + 2 request + finished), got %d", len(events))
	}
	if events[0].Name != constant.RunnerEventStarted {
		t.Fatalf("first event should be started, got %q", events[0].Name)
	}
	if events[len(events)-1].Name != constant.RunnerEventFinished {
		t.Fatalf("last event should be finished, got %q", events[len(events)-1].Name)
	}
	requestEventCount := 0
	for _, e := range events {
		if e.Name == constant.RunnerEventRequestDone {
			requestEventCount++
		}
	}
	if requestEventCount != 2 {
		t.Fatalf("want 2 request events, got %d", requestEventCount)
	}

	// Reports
	if raw, err := service.MarshalRunnerRunDetailJSON(detail); err != nil {
		t.Fatalf("json report: %v", err)
	} else if !strings.Contains(string(raw), `"passed_count": 2`) {
		t.Fatalf("json report does not reflect totals: %s", raw)
	}
	md := string(service.MarshalRunnerRunDetailMarkdown(detail))
	for _, want := range []string{"Auth flow", "01 Login", "02 Me", "PASS", "`environment.TOKEN`"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown report missing %q\n----\n%s", want, md)
		}
	}
}

func findRow(rows []entity.RunnerRunRequestRow, name string) *entity.RunnerRunRequestRow {
	for i := range rows {
		if rows[i].RequestName == name {
			return &rows[i]
		}
	}
	return nil
}

func ptr[T any](v T) *T { return &v }
