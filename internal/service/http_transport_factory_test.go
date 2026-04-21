package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"PostmanJanai/internal/constant"
)

type memSettings map[string]string

func (m memSettings) Get(ctx context.Context, key string) (string, error) {
	return m[key], nil
}

func TestManualProxy_NoProxy_IPWithoutPort_MatchesHostWithPort(t *testing.T) {
	proxyURL, err := url.Parse("http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	pf := manualProxy(proxyURL, "127.0.0.1")

	u, err := url.Parse("http://127.0.0.1:50114/direct")
	if err != nil {
		t.Fatal(err)
	}
	req := &http.Request{URL: u}
	got, err := pf(req)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected direct (nil proxy), got %v", got)
	}
}

func TestHTTPTransportFactory_ManualMode_NoProxyBypassesProxy(t *testing.T) {
	ctx := context.Background()
	cipher, err := NewSecretCipher()
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}

	proxySrv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("proxy should not be used")
	}))
	t.Cleanup(proxySrv.Close)

	pu, err := url.Parse(proxySrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	settings := memSettings{
		constant.SettingKeyProxyMode:    constant.ProxyModeManual,
		constant.SettingKeyProxyURL:     pu.String(),
		constant.SettingKeyProxyNoProxy: "127.0.0.1",
	}

	tf := &HTTPTransportFactory{Settings: settings, Cipher: cipher}
	tr, err := tf.Build(ctx, false)
	if err != nil {
		t.Fatal(err)
	}

	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(be.Close)

	u, err := url.Parse(be.URL + "/x")
	if err != nil {
		t.Fatal(err)
	}
	req := &http.Request{URL: u}
	puOut, err := tr.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if puOut != nil {
		t.Fatalf("expected direct connection, got proxy %v", puOut)
	}
}
