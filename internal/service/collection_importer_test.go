package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectJSONFormat(t *testing.T) {
	cases := []struct {
		name string
		body string
		want importFormatKind
	}{
		{"postman v2.1 schema", `{"info":{"name":"x","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`, formatPostmanV21},
		{"postman v2.0 schema", `{"info":{"name":"x","schema":"https://schema.getpostman.com/json/collection/v2.0.0/collection.json"},"item":[]}`, formatPostmanV20},
		{"postman v2.1 implied by item", `{"info":{"name":"x"},"item":[{"name":"a"}]}`, formatPostmanV21},
		{"postman v2.0 legacy", `{"name":"x","requests":[{"id":"r","method":"GET","url":"http://x"}]}`, formatPostmanV20},
		{"openapi 3", `{"openapi":"3.0.1","info":{"title":"x"},"paths":{"/a":{}}}`, formatOpenAPI3},
		{"swagger 2 unsupported", `{"swagger":"2.0","info":{"title":"x"}}`, formatUnknown},
		{"insomnia v4", `{"_type":"export","__export_format":4,"resources":[]}`, formatInsomniaV4},
		{"unknown", `{"hello":"world"}`, formatUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectJSONFormat([]byte(tc.body)); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestImportCollectionFromFile_Dispatch(t *testing.T) {
	dir := t.TempDir()
	// Postman v2.1
	p21 := filepath.Join(dir, "v21.json")
	if err := os.WriteFile(p21, []byte(postmanV21Sample), 0o600); err != nil {
		t.Fatal(err)
	}
	col, err := ImportCollectionFromFile(p21)
	if err != nil {
		t.Fatalf("v2.1 dispatch: %v", err)
	}
	if col.FormatLabel != "postman_v2.1" {
		t.Errorf("format mismatch: %s", col.FormatLabel)
	}
	// OpenAPI YAML
	yml := filepath.Join(dir, "api.yaml")
	if err := os.WriteFile(yml, []byte(openAPIYAMLSample), 0o600); err != nil {
		t.Fatal(err)
	}
	col2, err := ImportCollectionFromFile(yml)
	if err != nil {
		t.Fatalf("yaml dispatch: %v", err)
	}
	if col2.FormatLabel != "openapi_3.x" {
		t.Errorf("format mismatch: %s", col2.FormatLabel)
	}
	// Unknown
	unk := filepath.Join(dir, "weird.json")
	if err := os.WriteFile(unk, []byte(`{"hello":"world"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportCollectionFromFile(unk); err == nil || !strings.Contains(err.Error(), "Unsupported") {
		t.Errorf("expected unsupported format error, got %v", err)
	}
}

func TestImportCollectionFromFile_Empty(t *testing.T) {
	dir := t.TempDir()
	emp := filepath.Join(dir, "e.json")
	if err := os.WriteFile(emp, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportCollectionFromFile(emp); err == nil {
		t.Error("expected error for empty file")
	}
}
