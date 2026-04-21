package service

import (
	"PostmanJanai/internal/constant"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// SettingsKV is a minimal key/value reader for proxy settings (implemented by repository.SettingsRepository).
type SettingsKV interface {
	Get(ctx context.Context, key string) (string, error)
}

// TrustedCAPEMProvider returns PEM bytes for enabled custom CAs (implemented by repository.TrustedCARepository).
type TrustedCAPEMProvider interface {
	ListEnabledPEMs(ctx context.Context) ([][]byte, error)
}

// HTTPTransportFactory builds an http.Transport from persisted proxy + CA settings.
type HTTPTransportFactory struct {
	Settings SettingsKV
	CAs      TrustedCAPEMProvider
	Cipher   *SecretCipher
}

func (f *HTTPTransportFactory) tlsConfig(ctx context.Context, insecure bool) (*tls.Config, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if f != nil && f.CAs != nil {
		pems, err := f.CAs.ListEnabledPEMs(ctx)
		if err != nil {
			return nil, err
		}
		for _, pemBytes := range pems {
			if ok := pool.AppendCertsFromPEM(pemBytes); !ok {
				return nil, errors.New("failed to parse one or more custom CA certificates")
			}
		}
	}
	return &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: insecure,
		MinVersion:         tls.VersionTLS12,
	}, nil
}

func (f *HTTPTransportFactory) proxyFunc(ctx context.Context) (func(*http.Request) (*url.URL, error), error) {
	if f == nil || f.Settings == nil {
		return nil, nil
	}
	mode, err := f.Settings.Get(ctx, constant.SettingKeyProxyMode)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", constant.ProxyModeNone:
		return nil, nil
	case constant.ProxyModeSystem:
		return http.ProxyFromEnvironment, nil
	case constant.ProxyModeManual:
		rawURL, err := f.Settings.Get(ctx, constant.SettingKeyProxyURL)
		if err != nil {
			return nil, err
		}
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			return nil, nil
		}
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}
		user, _ := f.Settings.Get(ctx, constant.SettingKeyProxyUser)
		user = strings.TrimSpace(user)
		passEnc, _ := f.Settings.Get(ctx, constant.SettingKeyProxyPassword)
		var pass string
		if strings.TrimSpace(passEnc) != "" && f.Cipher != nil {
			p, err := f.Cipher.Decrypt(passEnc)
			if err != nil {
				return nil, err
			}
			pass = p
		}
		if user != "" || pass != "" {
			u.User = url.UserPassword(user, pass)
		}
		noProxy, _ := f.Settings.Get(ctx, constant.SettingKeyProxyNoProxy)
		return manualProxy(u, noProxy), nil
	default:
		return nil, nil
	}
}

// manualProxy returns a Proxy function honouring NO_PROXY for a fixed proxy base URL.
func manualProxy(proxy *url.URL, noProxy string) func(*http.Request) (*url.URL, error) {
	np := parseNoProxyList(noProxy)
	return func(req *http.Request) (*url.URL, error) {
		host := req.URL.Hostname()
		if host == "" {
			// If we can't determine a host, do not force traffic through the manual proxy.
			// (Some edge transports may omit Host before mapping; direct is safer than proxying blindly.)
			return nil, nil
		}
		for _, m := range np {
			if m.match(host) {
				return nil, nil
			}
		}
		return proxy, nil
	}
}

type noProxyEntry struct {
	raw    string
	host   string
	suffix bool // leading '.' means suffix match
}

func parseNoProxyList(s string) []noProxyEntry {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []noProxyEntry
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		e := noProxyEntry{raw: t}
		if strings.HasPrefix(t, ".") {
			e.suffix = true
			e.host = strings.ToLower(strings.TrimPrefix(t, "."))
		} else {
			e.host = strings.ToLower(t)
		}
		out = append(out, e)
	}
	return out
}

func (e noProxyEntry) match(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if e.suffix {
		return h == e.host || strings.HasSuffix(h, "."+e.host)
	}
	// Exact match on hostname OR host:port (NO_PROXY commonly lists IPs without port, but
	// Go's req.URL.Host for non-default ports includes ":port").
	if h == e.host {
		return true
	}
	if hostOnly, _, err := net.SplitHostPort(h); err == nil {
		return strings.ToLower(strings.TrimSpace(hostOnly)) == e.host
	}
	return false
}

// Build returns a configured http.Transport. `insecure` maps to tls.Config.InsecureSkipVerify.
func (f *HTTPTransportFactory) Build(ctx context.Context, insecure bool) (*http.Transport, error) {
	tlsCfg, err := f.tlsConfig(ctx, insecure)
	if err != nil {
		return nil, err
	}
	pf, err := f.proxyFunc(ctx)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		Proxy:             pf,
		TLSClientConfig:   tlsCfg,
		ForceAttemptHTTP2: true,
	}
	return tr, nil
}
