package service

import (
	"PostmanJanai/internal/entity"
	"testing"
)

func TestMergeAuthIntoHeadersAndQuery_Bearer(t *testing.T) {
	in := &entity.HTTPExecuteInput{
		Headers: []entity.KeyValue{{Key: "Authorization", Value: "old"}},
		Auth:    &entity.RequestAuth{Type: "bearer", BearerToken: "tok"},
	}
	MergeAuthIntoHeadersAndQuery(in)
	if len(in.Headers) != 1 || in.Headers[0].Value != "Bearer tok" {
		t.Fatalf("headers: %+v", in.Headers)
	}
}

func TestMergeAuthIntoHeadersAndQuery_APIKeyQuery(t *testing.T) {
	in := &entity.HTTPExecuteInput{
		QueryParams: []entity.KeyValue{{Key: "api_key", Value: "x"}},
		Auth:        &entity.RequestAuth{Type: "apikey", APIKeyName: "api_key", APIKey: "secret", APIKeyIn: "query"},
	}
	MergeAuthIntoHeadersAndQuery(in)
	if len(in.QueryParams) != 1 || in.QueryParams[0].Value != "secret" {
		t.Fatalf("query: %+v", in.QueryParams)
	}
}

func TestCloneSubstituteHTTPExecuteInput_AuthFields(t *testing.T) {
	vars := map[string]string{"t": "HELLO"}
	in := &entity.HTTPExecuteInput{
		Auth: &entity.RequestAuth{Type: "bearer", BearerToken: "{{ t }}"},
	}
	out := CloneSubstituteHTTPExecuteInput(in, vars)
	if out.Auth == nil || out.Auth.BearerToken != "HELLO" {
		t.Fatalf("got %+v", out.Auth)
	}
}
