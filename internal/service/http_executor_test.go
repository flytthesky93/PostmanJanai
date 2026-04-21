package service

import (
	"PostmanJanai/internal/entity"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPExecutor_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("want GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ex := NewHTTPExecutor(nil)
	res, err := ex.Execute(context.Background(), &entity.HTTPExecuteInput{
		Method:   "GET",
		URL:      srv.URL,
		BodyMode: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ErrorMessage != "" {
		t.Fatalf("transport error: %s", res.ErrorMessage)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if res.ResponseBody != "ok" {
		t.Fatalf("body %q", res.ResponseBody)
	}
}

func TestHTTPExecutor_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("x") != "1" || q.Get("y") != "two" {
			t.Errorf("query %#v", q)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ex := NewHTTPExecutor(nil)
	res, err := ex.Execute(context.Background(), &entity.HTTPExecuteInput{
		Method: "GET",
		URL:    srv.URL + "/path",
		QueryParams: []entity.KeyValue{
			{Key: "x", Value: "1"},
			{Key: "y", Value: "two"},
		},
		BodyMode: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalURL == "" || !strings.Contains(res.FinalURL, "x=1") || !strings.Contains(res.FinalURL, "y=two") {
		t.Fatalf("final URL missing query: %q", res.FinalURL)
	}
}

func TestHTTPExecutor_POST_rawJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"a":1}` {
			t.Errorf("body %s", b)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ex := NewHTTPExecutor(nil)
	res, err := ex.Execute(context.Background(), &entity.HTTPExecuteInput{
		Method:   "POST",
		URL:      srv.URL,
		BodyMode: "raw",
		Body:     `{"a":1}`,
		Headers:  []entity.KeyValue{{Key: "Content-Type", Value: "application/json"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !strings.Contains(res.ResponseBody, "ok") {
		t.Fatalf("response %q", res.ResponseBody)
	}
}

func TestHTTPExecutor_POST_xml(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if string(b) != `<note><to>x</to></note>` {
			t.Errorf("body %s", b)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/xml" {
			t.Errorf("Content-Type: want application/xml, got %q", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	defer srv.Close()

	ex := NewHTTPExecutor(nil)
	res, err := ex.Execute(context.Background(), &entity.HTTPExecuteInput{
		Method:   "POST",
		URL:      srv.URL,
		BodyMode: "xml",
		Body:     `<note><to>x</to></note>`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ErrorMessage != "" {
		t.Fatalf("transport: %s", res.ErrorMessage)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestHTTPExecutor_FormURLEncoded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("foo") != "bar" || r.FormValue("n") != "2" {
			t.Errorf("form %#v", r.Form)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
			t.Errorf("Content-Type: %s", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ex := NewHTTPExecutor(nil)
	res, err := ex.Execute(context.Background(), &entity.HTTPExecuteInput{
		Method:   "POST",
		URL:      srv.URL,
		BodyMode: "form_urlencoded",
		FormFields: []entity.KeyValue{
			{Key: "foo", Value: "bar"},
			{Key: "n", Value: "2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestHTTPExecutor_Multipart_textAndFile(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "data.txt")
	if err := os.WriteFile(filePath, []byte("file-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("title") != "hello" {
			t.Errorf("title %q", r.FormValue("title"))
		}
		f, hdr, err := r.FormFile("upload")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		data, _ := io.ReadAll(f)
		if string(data) != "file-bytes" {
			t.Fatalf("file content %q", data)
		}
		if hdr.Filename != "data.txt" {
			t.Errorf("filename %q", hdr.Filename)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	}))
	defer srv.Close()

	ex := NewHTTPExecutor(nil)
	res, err := ex.Execute(context.Background(), &entity.HTTPExecuteInput{
		Method:   "POST",
		URL:      srv.URL,
		BodyMode: "multipart",
		MultipartParts: []entity.MultipartPart{
			{Key: "title", Kind: "text", Value: "hello"},
			{Key: "upload", Kind: "file", FilePath: filePath},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ErrorMessage != "" {
		t.Fatalf("transport: %s", res.ErrorMessage)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if res.ResponseBody != "done" {
		t.Fatalf("body %q", res.ResponseBody)
	}
}

func TestHTTPExecutor_NilInput(t *testing.T) {
	ex := NewHTTPExecutor(nil)
	_, err := ex.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHTTPExecutor_InvalidURL(t *testing.T) {
	ex := NewHTTPExecutor(nil)
	_, err := ex.Execute(context.Background(), &entity.HTTPExecuteInput{
		Method: "GET",
		URL:    "not-a-url",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
