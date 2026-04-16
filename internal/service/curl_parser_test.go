package service

import (
	"PostmanJanai/internal/entity"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCurlCommand_SimpleGET(t *testing.T) {
	in, err := ParseCurlCommand(`curl https://httpbin.org/get`)
	if err != nil {
		t.Fatal(err)
	}
	if in.Method != "GET" {
		t.Fatalf("method %q", in.Method)
	}
	if in.URL != "https://httpbin.org/get" {
		t.Fatalf("url %q", in.URL)
	}
	if in.BodyMode != string(entity.BodyModeNone) {
		t.Fatalf("body mode %q", in.BodyMode)
	}
}

func TestParseCurlCommand_JSONBody(t *testing.T) {
	in, err := ParseCurlCommand(`curl -X POST https://httpbin.org/post -H "Content-Type: application/json" -d "{\"hello\":\"world\"}"`)
	if err != nil {
		t.Fatal(err)
	}
	if in.Method != "POST" {
		t.Fatalf("method %q", in.Method)
	}
	if in.BodyMode != string(entity.BodyModeRaw) {
		t.Fatalf("body mode %q want raw", in.BodyMode)
	}
	if !strings.Contains(in.Body, "hello") {
		t.Fatalf("body %q", in.Body)
	}
}

func TestParseCurlCommand_Form(t *testing.T) {
	in, err := ParseCurlCommand(`curl -X POST https://httpbin.org/post -d "foo=bar&n=2"`)
	if err != nil {
		t.Fatal(err)
	}
	if in.BodyMode != string(entity.BodyModeFormURLEncoded) {
		t.Fatalf("body mode %q", in.BodyMode)
	}
	if len(in.FormFields) < 2 {
		t.Fatalf("form fields %#v", in.FormFields)
	}
}

func TestParseCurlCommand_DataFromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "body.json")
	if err := os.WriteFile(p, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// Use forward slashes so shellwords does not treat \ as escape on Windows.
	pCurl := filepath.ToSlash(p)
	in, err := ParseCurlCommand(`curl -X POST https://httpbin.org/post -H "Content-Type: application/json" -d @` + pCurl)
	if err != nil {
		t.Fatal(err)
	}
	if in.BodyMode != string(entity.BodyModeRaw) {
		t.Fatalf("body mode %q", in.BodyMode)
	}
	if !strings.Contains(in.Body, `"x"`) {
		t.Fatalf("body from file: %q", in.Body)
	}
}

func TestParseCurlCommand_G_AppendsQuery(t *testing.T) {
	in, err := ParseCurlCommand(`curl -G https://httpbin.org/get -d "q=search" -d "page=2"`)
	if err != nil {
		t.Fatal(err)
	}
	if in.Method != "GET" {
		t.Fatalf("method %q", in.Method)
	}
	if !strings.Contains(in.URL, "q=search") || !strings.Contains(in.URL, "page=2") {
		t.Fatalf("url %q", in.URL)
	}
	if in.BodyMode != string(entity.BodyModeNone) || strings.TrimSpace(in.Body) != "" {
		t.Fatalf("expected no body, got mode %q body %q", in.BodyMode, in.Body)
	}
}

func TestParseCurlCommand_Multipart(t *testing.T) {
	in, err := ParseCurlCommand(`curl -X POST https://httpbin.org/post -F "title=hello" -F "note=there"`)
	if err != nil {
		t.Fatal(err)
	}
	if in.BodyMode != string(entity.BodyModeMultipartFormData) {
		t.Fatalf("body mode %q", in.BodyMode)
	}
	if len(in.MultipartParts) != 2 {
		t.Fatalf("parts %#v", in.MultipartParts)
	}
}

func TestParseCurlCommand_Empty(t *testing.T) {
	_, err := ParseCurlCommand("")
	if err == nil {
		t.Fatal("expected error")
	}
}
