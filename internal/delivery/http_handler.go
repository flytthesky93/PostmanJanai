package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/service"
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// HTTPHandler is the Wails binding for HTTP request execution (Phase 1).
type HTTPHandler struct {
	ctx      context.Context
	executor *service.HTTPExecutor
}

func NewHTTPHandler(ex *service.HTTPExecutor) *HTTPHandler {
	return &HTTPHandler{executor: ex}
}

func (h *HTTPHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *HTTPHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

// Execute runs an HTTP request from the UI payload.
func (h *HTTPHandler) Execute(in *entity.HTTPExecuteInput) (*entity.HTTPExecuteResult, error) {
	ctx := h.getContext()
	if in != nil {
		logger.D().InfoContext(ctx, "HTTPHandler.Execute", "method", in.Method, "url", in.URL)
	} else {
		logger.D().InfoContext(ctx, "HTTPHandler.Execute", "payload", nil)
	}
	res, err := h.executor.Execute(ctx, in)
	if err != nil {
		logger.L().ErrorContext(ctx, "HTTP execute validation failed", "error", err)
		return nil, err
	}
	if res.ErrorMessage != "" {
		logger.L().InfoContext(ctx, "HTTP execute finished with transport error", "error", res.ErrorMessage)
	} else {
		logger.L().InfoContext(ctx, "HTTP execute success", "status", res.StatusCode, "duration_ms", res.DurationMs)
	}
	return res, nil
}

// PickFileForBody opens a native file picker for multipart file fields. Returns empty if cancelled.
func (h *HTTPHandler) PickFileForBody() (string, error) {
	ctx := h.getContext()
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select file",
		Filters: []runtime.FileFilter{
			{DisplayName: "All files", Pattern: "*"},
		},
	})
}
