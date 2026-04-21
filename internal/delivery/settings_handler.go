package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/usecase"
	"context"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SettingsHandler exposes proxy / custom CA settings to the Wails frontend (Phase 6).
type SettingsHandler struct {
	ctx context.Context
	uc  usecase.SettingsUsecase
}

func NewSettingsHandler(uc usecase.SettingsUsecase) *SettingsHandler {
	return &SettingsHandler{uc: uc}
}

func (h *SettingsHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *SettingsHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

func (h *SettingsHandler) GetProxySettings() (*entity.ProxySettings, error) {
	ctx := h.getContext()
	return h.uc.GetProxySettings(ctx)
}

func (h *SettingsHandler) SetProxySettings(in *entity.ProxySettings) error {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "SettingsHandler.SetProxySettings", "mode", in.Mode)
	return h.uc.SetProxySettings(ctx, in)
}

func (h *SettingsHandler) TestProxy(targetURL string) (*entity.ProxyTestResult, error) {
	ctx := h.getContext()
	return h.uc.TestProxy(ctx, targetURL)
}

func (h *SettingsHandler) ListTrustedCAs() ([]entity.TrustedCASummary, error) {
	ctx := h.getContext()
	return h.uc.ListTrustedCAs(ctx)
}

func (h *SettingsHandler) AddTrustedCA(label, pemContent string) error {
	ctx := h.getContext()
	return h.uc.AddTrustedCA(ctx, label, pemContent)
}

func (h *SettingsHandler) SetTrustedCAEnabled(id string, enabled bool) error {
	ctx := h.getContext()
	return h.uc.SetTrustedCAEnabled(ctx, id, enabled)
}

func (h *SettingsHandler) DeleteTrustedCA(id string) error {
	ctx := h.getContext()
	return h.uc.DeleteTrustedCA(ctx, id)
}

// PickCACertFile opens a native file picker for a PEM-encoded certificate.
func (h *SettingsHandler) PickCACertFile() (string, error) {
	ctx := h.getContext()
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select CA certificate",
		Filters: []runtime.FileFilter{
			{DisplayName: "Certificates", Pattern: "*.pem;*.crt;*.cer"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
}

// ReadTextFile reads a small UTF-8 text file from disk (used after PickCACertFile).
func (h *SettingsHandler) ReadTextFile(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
