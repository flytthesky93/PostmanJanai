package service

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"testing"
)

func TestRunAssertionRules_StatusAndJSON(t *testing.T) {
	body := `{"name":"Alice","age":30,"items":[1,2,3]}`
	cap := NewCaptureContext(200, []entity.KeyValue{{Key: "Content-Type", Value: "application/json"}}, body)
	ctx := AssertionContextFromCapture(cap, 120, 64)

	rules := []entity.RequestAssertionRow{
		{Name: "status eq", Source: constant.AssertionSourceStatus, Operator: constant.AssertionOpEq, Expected: "200", Enabled: true},
		{Name: "name eq", Source: constant.AssertionSourceJSONBody, Expression: "$.name", Operator: constant.AssertionOpEq, Expected: "Alice", Enabled: true},
		{Name: "age gt", Source: constant.AssertionSourceJSONBody, Expression: "$.age", Operator: constant.AssertionOpGT, Expected: "18", Enabled: true},
		{Name: "items length lt", Source: constant.AssertionSourceJSONBody, Expression: "$.items.length", Operator: constant.AssertionOpLT, Expected: "10", Enabled: true},
		{Name: "header exists", Source: constant.AssertionSourceHeader, Expression: "Content-Type", Operator: constant.AssertionOpExists, Enabled: true},
		{Name: "header missing", Source: constant.AssertionSourceHeader, Expression: "X-Missing", Operator: constant.AssertionOpNotExists, Enabled: true},
		{Name: "duration lt 1s", Source: constant.AssertionSourceDurationMs, Operator: constant.AssertionOpLT, Expected: "1000", Enabled: true},
		{Name: "size gte", Source: constant.AssertionSourceResponseSizeBytes, Operator: constant.AssertionOpGTE, Expected: "1", Enabled: true},
		{Name: "body contains", Source: constant.AssertionSourceRegexBody, Expression: `name`, Operator: constant.AssertionOpExists, Enabled: true},
	}

	res := RunAssertionRules(ctx, rules)
	if len(res) != len(rules) {
		t.Fatalf("expected %d results, got %d", len(rules), len(res))
	}
	for _, r := range res {
		if !r.Passed {
			t.Errorf("assertion %q failed: actual=%q err=%q", r.Name, r.Actual, r.ErrorMessage)
		}
	}
}

func TestRunAssertionRules_FailureCase(t *testing.T) {
	cap := NewCaptureContext(404, nil, "")
	ctx := AssertionContextFromCapture(cap, 50, 0)
	res := RunAssertionRules(ctx, []entity.RequestAssertionRow{
		{Name: "status 200", Source: constant.AssertionSourceStatus, Operator: constant.AssertionOpEq, Expected: "200", Enabled: true},
	})
	if len(res) != 1 || res[0].Passed {
		t.Fatalf("expected single failing assertion, got %+v", res)
	}
	if res[0].Actual != "404" {
		t.Errorf("actual = %q, want 404", res[0].Actual)
	}
}

func TestRunAssertionRules_RegexOperator(t *testing.T) {
	cap := NewCaptureContext(200, nil, `{"id":"USR-1234"}`)
	ctx := AssertionContextFromCapture(cap, 0, 0)
	res := RunAssertionRules(ctx, []entity.RequestAssertionRow{
		{Name: "id pattern", Source: constant.AssertionSourceJSONBody, Expression: "$.id", Operator: constant.AssertionOpRegex, Expected: `^USR-\d+$`, Enabled: true},
	})
	if len(res) != 1 || !res[0].Passed {
		t.Fatalf("expected pass, got %+v", res)
	}
}
