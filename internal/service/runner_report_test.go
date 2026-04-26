package service

import (
	"PostmanJanai/internal/entity"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sampleDetail() *entity.RunnerRunDetail {
	finish := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return &entity.RunnerRunDetail{
		RunnerRunSummary: entity.RunnerRunSummary{
			ID:              "run-1",
			FolderName:      "My folder",
			EnvironmentName: "dev",
			Status:          "completed",
			TotalCount:      2,
			PassedCount:     1,
			FailedCount:     1,
			ErrorCount:      0,
			DurationMs:      123,
			StartedAt:       time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC),
			FinishedAt:      &finish,
		},
		Notes: "smoke check",
		Requests: []entity.RunnerRunRequestRow{
			{
				ID:                "row-1",
				RequestName:       "Login",
				Method:            "POST",
				URL:               "https://api/login",
				Status:            "passed",
				StatusCode:        200,
				DurationMs:        50,
				ResponseSizeBytes: 12,
				Assertions: []entity.AssertionResult{
					{Name: "status ok", Source: "status", Operator: "eq", Expected: "200", Actual: "200", Passed: true},
				},
				Captures: []entity.CaptureResult{
					{Name: "token", Source: "json_body", Expression: "$.token", TargetScope: "environment", TargetVariable: "TOKEN", Value: "abc", Captured: true},
				},
			},
			{
				ID:           "row-2",
				RequestName:  "Get me",
				Method:       "GET",
				URL:          "https://api/me|raw",
				Status:       "failed",
				StatusCode:   500,
				DurationMs:   80,
				ErrorMessage: "boom",
				Assertions: []entity.AssertionResult{
					{Name: "is 200", Source: "status", Operator: "eq", Expected: "200", Actual: "500", Passed: false, ErrorMessage: "expected 200, got 500"},
				},
			},
		},
	}
}

func TestRunnerReport_JSONRoundTrip(t *testing.T) {
	data, err := MarshalRunnerRunDetailJSON(sampleDetail())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back entity.RunnerRunDetail
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.ID != "run-1" || back.PassedCount != 1 || back.FailedCount != 1 {
		t.Fatalf("round-trip lost summary fields: %+v", back.RunnerRunSummary)
	}
	if len(back.Requests) != 2 {
		t.Fatalf("requests lost: %d", len(back.Requests))
	}
}

func TestRunnerReport_MarkdownContainsKeySignals(t *testing.T) {
	md := string(MarshalRunnerRunDetailMarkdown(sampleDetail()))
	mustContain := []string{
		"# Runner report",
		"My folder",
		"`completed`",
		"smoke check",
		"## Requests",
		"## Details",
		"PASS",
		"FAIL",
		"`token`",
		"`environment.TOKEN`",
		// pipe escaped so the row stays a single column
		`https://api/me\|raw`,
	}
	for _, want := range mustContain {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q\n----\n%s", want, md)
		}
	}
}

func TestRunnerReport_NilSafety(t *testing.T) {
	if _, err := MarshalRunnerRunDetailJSON(nil); err != nil {
		t.Fatalf("nil json: %v", err)
	}
	md := MarshalRunnerRunDetailMarkdown(nil)
	if !strings.Contains(string(md), "No data") {
		t.Fatalf("nil markdown should be safe placeholder, got %q", string(md))
	}
}
