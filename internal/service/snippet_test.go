package service

import (
	"PostmanJanai/internal/entity"
	"strings"
	"testing"
)

func TestRenderSnippet_CurlBashGET(t *testing.T) {
	in := &entity.HTTPExecuteInput{
		Method: "GET",
		URL:    "https://example.com/api",
		QueryParams: []entity.KeyValue{
			{Key: "q", Value: "hello"},
		},
	}
	s, err := RenderSnippet(in, SnippetKindCurlBash)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "curl") || !strings.Contains(s, "example.com") {
		t.Fatalf("unexpected: %q", s)
	}
}

func TestFinalURLForRequest_QueryMerge(t *testing.T) {
	in := &entity.HTTPExecuteInput{
		URL: "https://example.com/path?a=1",
		QueryParams: []entity.KeyValue{
			{Key: "b", Value: "2"},
		},
	}
	u, err := FinalURLForRequest(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "a=1") || !strings.Contains(u, "b=2") {
		t.Fatalf("want merged query, got %q", u)
	}
}
