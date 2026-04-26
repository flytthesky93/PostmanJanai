package entity

import "time"

// RunnerRunSummary — header row for recent runs lists.
type RunnerRunSummary struct {
	ID              string     `json:"id"`
	FolderID        *string    `json:"folder_id,omitempty"`
	FolderName      string     `json:"folder_name"`
	EnvironmentID   *string    `json:"environment_id,omitempty"`
	EnvironmentName string     `json:"environment_name"`
	Status          string     `json:"status"`
	TotalCount      int        `json:"total_count"`
	PassedCount     int        `json:"passed_count"`
	FailedCount     int        `json:"failed_count"`
	ErrorCount      int        `json:"error_count"`
	DurationMs      int        `json:"duration_ms"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

// RunnerRunRequestRow — one request result within a run (for detail view).
//
// Phase 8.1: the resolved request (post {{var}} substitution) and the response
// payload are persisted alongside the existing metrics so the user can inspect
// the round trip later. Body fields use omitempty so the live progress event
// stream doesn't ship empty placeholders for skipped/errored rows.
type RunnerRunRequestRow struct {
	ID                  string            `json:"id"`
	RunID               string            `json:"run_id"`
	RequestID           *string           `json:"request_id,omitempty"`
	RequestName         string            `json:"request_name"`
	Method              string            `json:"method"`
	URL                 string            `json:"url"`
	Status              string            `json:"status"`
	StatusCode          int               `json:"status_code"`
	DurationMs          int               `json:"duration_ms"`
	ResponseSizeBytes   int               `json:"response_size_bytes"`
	ErrorMessage        string            `json:"error_message,omitempty"`
	RequestHeadersJSON  string            `json:"request_headers_json,omitempty"`
	ResponseHeadersJSON string            `json:"response_headers_json,omitempty"`
	RequestBody         string            `json:"request_body,omitempty"`
	ResponseBody        string            `json:"response_body,omitempty"`
	BodyTruncated       bool              `json:"body_truncated,omitempty"`
	Assertions          []AssertionResult `json:"assertions,omitempty"`
	Captures            []CaptureResult   `json:"captures,omitempty"`
	SortOrder           int               `json:"sort_order"`
	CreatedAt           time.Time         `json:"created_at"`
}

// RunnerRunDetail — run header + per-request results.
type RunnerRunDetail struct {
	RunnerRunSummary
	Requests []RunnerRunRequestRow `json:"requests"`
	Notes    string                `json:"notes"`
}

// RunFolderInput — payload from the UI to start a folder run.
//
// Iterations / DelayMs / TimeoutPerRequestMs were promised in Phase 8 but
// landed in 8.1 (this iteration). Defaults preserve previous behavior:
//   - Iterations          ≤0 → 1 iteration; clamped to RunnerMaxIterations.
//   - DelayMs             ≤0 → no delay; clamped to RunnerMaxDelayMs.
//   - TimeoutPerRequestMs ≤0 → no per-request override (uses HTTPClientTimeout);
//                              otherwise wraps each Execute call in its own
//                              context.WithTimeout so a hung request can't
//                              starve the rest of the plan.
type RunFolderInput struct {
	FolderID             string `json:"folder_id"`
	EnvironmentID        string `json:"environment_id,omitempty"`
	StopOnFail           bool   `json:"stop_on_fail,omitempty"`
	Notes                string `json:"notes,omitempty"`
	Iterations           int    `json:"iterations,omitempty"`
	DelayMs              int    `json:"delay_ms,omitempty"`
	TimeoutPerRequestMs  int    `json:"timeout_per_request_ms,omitempty"`
}

// RunnerProgressEvent — emitted to the frontend as the runner advances.
type RunnerProgressEvent struct {
	RunID       string  `json:"run_id"`
	TotalCount  int     `json:"total_count"`
	CurrentIdx  int     `json:"current_idx"`
	PassedCount int     `json:"passed_count"`
	FailedCount int     `json:"failed_count"`
	ErrorCount  int     `json:"error_count"`
	Phase       string  `json:"phase"`
	Status      string  `json:"status,omitempty"`
	Request     *RunnerRunRequestRow `json:"request,omitempty"`
}
