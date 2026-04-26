package service

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"testing"
)

func TestRunCaptureRules_AllSources(t *testing.T) {
	body := `{"token":"abc123","items":[1,2,3]}`
	ctx := NewCaptureContext(200, []entity.KeyValue{
		{Key: "X-Trace-Id", Value: "trace-42"},
		{Key: "Content-Type", Value: "application/json"},
	}, body)

	rules := []entity.RequestCaptureRow{
		{Name: "status", Source: constant.CaptureSourceStatus, TargetVariable: "status", TargetScope: constant.CaptureScopeEnvironment, Enabled: true},
		{Name: "trace", Source: constant.CaptureSourceHeader, Expression: "x-trace-id", TargetVariable: "trace_id", TargetScope: constant.CaptureScopeEnvironment, Enabled: true},
		{Name: "token", Source: constant.CaptureSourceJSONBody, Expression: "$.token", TargetVariable: "auth_token", TargetScope: constant.CaptureScopeEnvironment, Enabled: true},
		{Name: "first", Source: constant.CaptureSourceJSONBody, Expression: "$.items[0]", TargetVariable: "first", TargetScope: constant.CaptureScopeEnvironment, Enabled: true},
		{Name: "regex", Source: constant.CaptureSourceRegexBody, Expression: `"token":"([^"]+)"`, TargetVariable: "regex_token", TargetScope: constant.CaptureScopeEnvironment, Enabled: true},
		{Name: "disabled", Source: constant.CaptureSourceStatus, TargetVariable: "x", TargetScope: constant.CaptureScopeEnvironment, Enabled: false},
	}
	res := RunCaptureRules(ctx, rules)

	if len(res) != 5 {
		t.Fatalf("expected 5 results (disabled rule skipped), got %d", len(res))
	}
	if !res[0].Captured || res[0].Value != "200" {
		t.Errorf("status: %+v", res[0])
	}
	if !res[1].Captured || res[1].Value != "trace-42" {
		t.Errorf("header: %+v", res[1])
	}
	if !res[2].Captured || res[2].Value != "abc123" {
		t.Errorf("json: %+v", res[2])
	}
	if !res[3].Captured || res[3].Value != "1" {
		t.Errorf("json arr: %+v", res[3])
	}
	if !res[4].Captured || res[4].Value != "abc123" {
		t.Errorf("regex: %+v", res[4])
	}
}

func TestRunCaptureRules_MissingHeaderProducesError(t *testing.T) {
	ctx := NewCaptureContext(200, nil, "")
	res := RunCaptureRules(ctx, []entity.RequestCaptureRow{
		{Name: "x", Source: constant.CaptureSourceHeader, Expression: "X-Missing", TargetVariable: "x", Enabled: true},
	})
	if len(res) != 1 || res[0].Captured {
		t.Fatalf("expected one failed capture, got %+v", res)
	}
	if res[0].ErrorMessage == "" {
		t.Fatalf("expected error message, got empty")
	}
}

func TestRunCaptureRules_InvalidJSONBody(t *testing.T) {
	ctx := NewCaptureContext(200, nil, "not json")
	res := RunCaptureRules(ctx, []entity.RequestCaptureRow{
		{Name: "json", Source: constant.CaptureSourceJSONBody, Expression: "$.x", TargetVariable: "x", Enabled: true},
	})
	if len(res) != 1 || res[0].Captured {
		t.Fatalf("expected failed capture, got %+v", res)
	}
}
