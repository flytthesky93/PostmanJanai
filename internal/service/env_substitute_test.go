package service

import (
	"PostmanJanai/internal/entity"
	"testing"
)

func TestSubstituteEnvVars(t *testing.T) {
	vars := map[string]string{"host": "example.com", "empty": ""}
	if got := SubstituteEnvVars("https://{{ host }}/x", vars); got != "https://example.com/x" {
		t.Fatalf("got %q", got)
	}
	if got := SubstituteEnvVars("{{missing}}", vars); got != "" {
		t.Fatalf("unknown should be empty, got %q", got)
	}
	if got := SubstituteEnvVars("a{{empty}}b", vars); got != "ab" {
		t.Fatalf("empty value in map: got %q", got)
	}
}

func TestCloneSubstituteHTTPExecuteInput_SubstitutesKeysAndMultipartPath(t *testing.T) {
	vars := map[string]string{"k": "key1", "p": "/tmp/a.txt"}
	in := &entity.HTTPExecuteInput{
		URL:         "https://x/{{k}}",
		QueryParams: []entity.KeyValue{{Key: "{{k}}", Value: "v"}},
		MultipartParts: []entity.MultipartPart{{
			Key: "f", Kind: "file", FilePath: "{{p}}",
		}},
	}
	out := CloneSubstituteHTTPExecuteInput(in, vars)
	if out.QueryParams[0].Key != "key1" || out.QueryParams[0].Value != "v" {
		t.Fatalf("query: %+v", out.QueryParams[0])
	}
	if out.MultipartParts[0].FilePath != "/tmp/a.txt" {
		t.Fatalf("file path: %q", out.MultipartParts[0].FilePath)
	}
}

func TestCloneSubstituteHTTPExecuteInput_SubstitutesBody(t *testing.T) {
	vars := map[string]string{"token": "abc123"}
	in := &entity.HTTPExecuteInput{
		Body: `{"x":"{{ token }}"}`,
	}
	out := CloneSubstituteHTTPExecuteInput(in, vars)
	if out.Body != `{"x":"abc123"}` {
		t.Fatalf("body: got %q", out.Body)
	}
}
