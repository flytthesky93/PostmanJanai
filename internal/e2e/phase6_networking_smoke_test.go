package e2e

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/service"
	"PostmanJanai/internal/testutil"
)

func TestPhase6_CustomCA_TLSWithoutInsecureSkipVerify(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	cipher, err := service.NewSecretCipher()
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	settings := repository.NewSettingsRepository(client)
	cas := repository.NewTrustedCARepository(client)
	tf := &service.HTTPTransportFactory{Settings: settings, CAs: cas, Cipher: cipher}
	ex := service.NewHTTPExecutor(tf)

	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("tls-ok"))
	}))
	t.Cleanup(backend.Close)

	// Without importing the httptest leaf cert, TLS should fail.
	resBad, err := ex.Execute(ctx, &entity.HTTPExecuteInput{
		Method:   http.MethodGet,
		URL:      backend.URL + "/ping",
		BodyMode: "none",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resBad.ErrorMessage == "" {
		t.Fatalf("expected TLS error without custom CA, got ok: %#v", resBad)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: backend.Certificate().Raw})
	if _, err := cas.Create(ctx, "httptest-leaf", string(certPEM)); err != nil {
		t.Fatalf("add ca: %v", err)
	}

	resOK, err := ex.Execute(ctx, &entity.HTTPExecuteInput{
		Method:   http.MethodGet,
		URL:      backend.URL + "/ping",
		BodyMode: "none",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resOK.ErrorMessage != "" {
		t.Fatalf("transport err: %s", resOK.ErrorMessage)
	}
	if resOK.StatusCode != http.StatusOK || resOK.ResponseBody != "tls-ok" {
		t.Fatalf("unexpected response: %#v", resOK)
	}
}

func TestPhase6_SystemProxy_HTTP_PROXY(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	cipher, err := service.NewSecretCipher()
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	settings := repository.NewSettingsRepository(client)
	tf := &service.HTTPTransportFactory{Settings: settings, CAs: nil, Cipher: cipher}
	ex := service.NewHTTPExecutor(tf)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hit-backend"))
	}))
	t.Cleanup(backend.Close)

	beURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(httputil.NewSingleHostReverseProxy(beURL))
	t.Cleanup(proxy.Close)

	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("HTTPS_PROXY", proxy.URL)

	if err := settings.Set(ctx, constant.SettingKeyProxyMode, constant.ProxyModeSystem); err != nil {
		t.Fatalf("set proxy mode: %v", err)
	}

	res, err := ex.Execute(ctx, &entity.HTTPExecuteInput{
		Method:   http.MethodGet,
		URL:      backend.URL + "/via-proxy",
		BodyMode: "none",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.ErrorMessage != "" {
		t.Fatalf("transport err: %s", res.ErrorMessage)
	}
	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("status %d, want %d", res.StatusCode, http.StatusTeapot)
	}
}

func TestPhase6_ManualProxy_WithBasicAuth(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	cipher, err := service.NewSecretCipher()
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	settings := repository.NewSettingsRepository(client)
	tf := &service.HTTPTransportFactory{Settings: settings, CAs: nil, Cipher: cipher}
	ex := service.NewHTTPExecutor(tf)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "proxied")
	}))
	t.Cleanup(backend.Close)
	beURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	rp := httputil.NewSingleHostReverseProxy(beURL)

	var proxyHits int
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("u:p"))
		if got := r.Header.Get("Proxy-Authorization"); got != want {
			http.Error(w, "proxy auth", http.StatusProxyAuthRequired)
			return
		}
		proxyHits++
		rp.ServeHTTP(w, r)
	}))
	t.Cleanup(proxy.Close)

	pu, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}

	passEnc, err := cipher.Encrypt("p")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := settings.Set(ctx, constant.SettingKeyProxyMode, constant.ProxyModeManual); err != nil {
		t.Fatalf("mode: %v", err)
	}
	if err := settings.Set(ctx, constant.SettingKeyProxyURL, pu.String()); err != nil {
		t.Fatalf("proxy url: %v", err)
	}
	if err := settings.Set(ctx, constant.SettingKeyProxyUser, "u"); err != nil {
		t.Fatalf("proxy user: %v", err)
	}
	if err := settings.Set(ctx, constant.SettingKeyProxyPassword, passEnc); err != nil {
		t.Fatalf("proxy pass: %v", err)
	}

	res, err := ex.Execute(ctx, &entity.HTTPExecuteInput{
		Method:   http.MethodGet,
		URL:      backend.URL + "/x",
		BodyMode: "none",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.ErrorMessage != "" {
		t.Fatalf("transport err: %s", res.ErrorMessage)
	}
	if res.StatusCode != http.StatusOK || res.ResponseBody != "proxied" {
		t.Fatalf("bad response: %#v", res)
	}
	if proxyHits == 0 {
		t.Fatalf("expected proxy to see request")
	}
}

func TestPhase6_ManualProxy_NO_PROXYBypass(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewEntClient(t)
	cipher, err := service.NewSecretCipher()
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	settings := repository.NewSettingsRepository(client)
	tf := &service.HTTPTransportFactory{Settings: settings, CAs: nil, Cipher: cipher}
	ex := service.NewHTTPExecutor(tf)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(backend.Close)

	var proxyHits int
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits++
		http.Error(w, "should not hit proxy", http.StatusBadGateway)
	}))
	t.Cleanup(proxy.Close)

	pu, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := settings.Set(ctx, constant.SettingKeyProxyMode, constant.ProxyModeManual); err != nil {
		t.Fatalf("mode: %v", err)
	}
	if err := settings.Set(ctx, constant.SettingKeyProxyURL, pu.String()); err != nil {
		t.Fatalf("proxy url: %v", err)
	}
	host := strings.TrimPrefix(backend.URL, "http://")
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	hostOnly := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostOnly = h
	}
	if err := settings.Set(ctx, constant.SettingKeyProxyNoProxy, hostOnly); err != nil {
		t.Fatalf("no_proxy: %v", err)
	}
	np, err := settings.Get(ctx, constant.SettingKeyProxyNoProxy)
	if err != nil {
		t.Fatalf("read no_proxy: %v", err)
	}
	if strings.TrimSpace(np) == "" {
		t.Fatalf("no_proxy not persisted")
	}
	res, err := ex.Execute(ctx, &entity.HTTPExecuteInput{
		Method:   http.MethodGet,
		URL:      backend.URL + "/direct",
		BodyMode: "none",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.ErrorMessage != "" {
		t.Fatalf("transport err: %s", res.ErrorMessage)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d body %q err %q", res.StatusCode, res.ResponseBody, res.ErrorMessage)
	}
	if proxyHits != 0 {
		t.Fatalf("expected NO_PROXY to bypass proxy, proxyHits=%d", proxyHits)
	}
}
