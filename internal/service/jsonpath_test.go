package service

import (
	"encoding/json"
	"testing"
)

func parseJSON(t *testing.T, raw string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return v
}

func TestEvalJSONPath_KeyAndIndex(t *testing.T) {
	v := parseJSON(t, `{"data":{"items":[{"id":1,"name":"a"},{"id":2,"name":"b"}]}}`)
	cases := []struct {
		path string
		want string
	}{
		{"$.data.items[0].id", "1"},
		{"data.items[1].name", "b"},
		{"$.data.items[-1].id", "2"},
		{"$.data.items.length", "2"},
	}
	for _, c := range cases {
		got, err := EvalJSONPath(c.path, v)
		if err != nil {
			t.Fatalf("path %q: %v", c.path, err)
		}
		if s := JSONValueToString(got); s != c.want {
			t.Errorf("path %q = %q, want %q", c.path, s, c.want)
		}
	}
}

func TestEvalJSONPath_BracketKey(t *testing.T) {
	v := parseJSON(t, `{"x.y":{"a":42}}`)
	got, err := EvalJSONPath(`$["x.y"].a`, v)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if s := JSONValueToString(got); s != "42" {
		t.Errorf("got %q want 42", s)
	}
}

func TestEvalJSONPath_StarReturnsSlice(t *testing.T) {
	v := parseJSON(t, `{"items":["a","b","c"]}`)
	got, err := EvalJSONPath(`$.items[*]`, v)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	arr, ok := got.([]any)
	if !ok || len(arr) != 3 {
		t.Fatalf("expected slice of 3, got %T %v", got, got)
	}
}

func TestEvalJSONPath_MissingKeyReturnsError(t *testing.T) {
	v := parseJSON(t, `{"foo":1}`)
	if _, err := EvalJSONPath(`$.bar`, v); err == nil {
		t.Fatalf("expected error for missing key")
	}
}

func TestJSONValueToString_Primitives(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"hi", "hi"},
		{true, "true"},
		{false, "false"},
		{float64(7), "7"},
		{float64(3.14), "3.14"},
	}
	for _, c := range cases {
		if got := JSONValueToString(c.in); got != c.want {
			t.Errorf("%v: got %q want %q", c.in, got, c.want)
		}
	}
}
