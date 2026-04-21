package usecase

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/service"
	"context"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"PostmanJanai/ent"
)

// SettingsUsecase manages proxy / custom CA settings (Phase 6).
type SettingsUsecase interface {
	GetProxySettings(ctx context.Context) (*entity.ProxySettings, error)
	SetProxySettings(ctx context.Context, in *entity.ProxySettings) error
	TestProxy(ctx context.Context, targetURL string) (*entity.ProxyTestResult, error)

	ListTrustedCAs(ctx context.Context) ([]entity.TrustedCASummary, error)
	AddTrustedCA(ctx context.Context, label, pemContent string) error
	SetTrustedCAEnabled(ctx context.Context, id string, enabled bool) error
	DeleteTrustedCA(ctx context.Context, id string) error
}

type settingsUsecaseImpl struct {
	settings repository.SettingsRepository
	cas      repository.TrustedCARepository
	cipher   *service.SecretCipher
}

func NewSettingsUsecase(settings repository.SettingsRepository, cas repository.TrustedCARepository, cipher *service.SecretCipher) SettingsUsecase {
	return &settingsUsecaseImpl{settings: settings, cas: cas, cipher: cipher}
}

func (u *settingsUsecaseImpl) GetProxySettings(ctx context.Context) (*entity.ProxySettings, error) {
	mode, _ := u.settings.Get(ctx, constant.SettingKeyProxyMode)
	rawURL, _ := u.settings.Get(ctx, constant.SettingKeyProxyURL)
	user, _ := u.settings.Get(ctx, constant.SettingKeyProxyUser)
	np, _ := u.settings.Get(ctx, constant.SettingKeyProxyNoProxy)
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = constant.ProxyModeNone
	}
	out := &entity.ProxySettings{
		Mode:     m,
		URL:      strings.TrimSpace(rawURL),
		Username: strings.TrimSpace(user),
		Password: "",
		NoProxy:  strings.TrimSpace(np),
	}
	return out, nil
}

func (u *settingsUsecaseImpl) SetProxySettings(ctx context.Context, in *entity.ProxySettings) error {
	if in == nil {
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	switch mode {
	case constant.ProxyModeNone, constant.ProxyModeSystem, constant.ProxyModeManual:
	default:
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	if err := u.settings.Set(ctx, constant.SettingKeyProxyMode, mode); err != nil {
		return err
	}
	if mode == constant.ProxyModeManual {
		uStr := strings.TrimSpace(in.URL)
		if uStr != "" {
			if _, err := url.Parse(uStr); err != nil {
				return apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
			}
		}
		if err := u.settings.Set(ctx, constant.SettingKeyProxyURL, uStr); err != nil {
			return err
		}
		if err := u.settings.Set(ctx, constant.SettingKeyProxyUser, strings.TrimSpace(in.Username)); err != nil {
			return err
		}
		if strings.TrimSpace(in.Password) != "" {
			if u.cipher == nil {
				return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
			}
			enc, err := u.cipher.Encrypt(strings.TrimSpace(in.Password))
			if err != nil {
				return apperror.NewWithErrorDetail(constant.ErrInternal, err)
			}
			if err := u.settings.Set(ctx, constant.SettingKeyProxyPassword, enc); err != nil {
				return err
			}
		}
		if err := u.settings.Set(ctx, constant.SettingKeyProxyNoProxy, strings.TrimSpace(in.NoProxy)); err != nil {
			return err
		}
	} else {
		_ = u.settings.Set(ctx, constant.SettingKeyProxyURL, "")
		_ = u.settings.Set(ctx, constant.SettingKeyProxyUser, "")
		_ = u.settings.Set(ctx, constant.SettingKeyProxyPassword, "")
		_ = u.settings.Set(ctx, constant.SettingKeyProxyNoProxy, "")
	}
	return nil
}

func (u *settingsUsecaseImpl) TestProxy(ctx context.Context, targetURL string) (*entity.ProxyTestResult, error) {
	raw := strings.TrimSpace(targetURL)
	if raw == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, nil)
	}
	if _, err := url.Parse(raw); err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
	}
	tf := &service.HTTPTransportFactory{Settings: u.settings, CAs: u.cas, Cipher: u.cipher}
	tr, err := tf.Build(ctx, false)
	if err != nil {
		return &entity.ProxyTestResult{OK: false, ErrorMessage: err.Error()}, nil
	}
	timeout := time.Duration(constant.ProxyTestTimeoutSeconds) * time.Second
	client := &http.Client{Timeout: timeout, Transport: tr}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return &entity.ProxyTestResult{OK: false, ErrorMessage: err.Error()}, nil
	}
	resp, err := client.Do(req)
	dur := time.Since(start).Milliseconds()
	if err != nil {
		return &entity.ProxyTestResult{OK: false, DurationMs: dur, ErrorMessage: err.Error()}, nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return &entity.ProxyTestResult{
		OK:         true,
		StatusCode: resp.StatusCode,
		DurationMs: dur,
		FinalURL:   raw,
	}, nil
}

func (u *settingsUsecaseImpl) ListTrustedCAs(ctx context.Context) ([]entity.TrustedCASummary, error) {
	rows, err := u.cas.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.TrustedCASummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, entity.TrustedCASummary{
			ID:        r.ID.String(),
			Label:     r.Label,
			Enabled:   r.Enabled,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}

func (u *settingsUsecaseImpl) AddTrustedCA(ctx context.Context, label, pemContent string) error {
	l := strings.TrimSpace(label)
	p := strings.TrimSpace(pemContent)
	if l == "" || p == "" {
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	block, _ := pem.Decode([]byte(p))
	if block == nil {
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return apperror.NewWithErrorDetail(constant.ErrInternal, err)
	}
	_, err := u.cas.Create(ctx, l, p)
	if err != nil {
		if ent.IsConstraintError(err) {
			return apperror.NewWithErrorDetail(constant.ErrInternal, err)
		}
		return apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	return nil
}

func (u *settingsUsecaseImpl) SetTrustedCAEnabled(ctx context.Context, id string, enabled bool) error {
	return u.cas.SetEnabled(ctx, id, enabled)
}

func (u *settingsUsecaseImpl) DeleteTrustedCA(ctx context.Context, id string) error {
	return u.cas.Delete(ctx, id)
}
