package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/service"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// HTTPHandler is the Wails binding for HTTP request execution (Phase 1).
type HTTPHandler struct {
	ctx      context.Context
	executor *service.HTTPExecutor
	history  repository.HistoryRepository
}

func NewHTTPHandler(ex *service.HTTPExecutor, hist repository.HistoryRepository) *HTTPHandler {
	return &HTTPHandler{executor: ex, history: hist}
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
	h.persistHistory(ctx, in, res)
	return res, nil
}

func (h *HTTPHandler) persistHistory(ctx context.Context, in *entity.HTTPExecuteInput, res *entity.HTTPExecuteResult) {
	if h.history == nil || res == nil {
		return
	}

	method := http.MethodGet
	if in != nil {
		if m := strings.TrimSpace(strings.ToUpper(in.Method)); m != "" {
			method = m
		}
	}
	urlStr := strings.TrimSpace(res.FinalURL)
	if urlStr == "" && in != nil {
		urlStr = strings.TrimSpace(in.URL)
	}

	var reqHdrJSON *string
	if len(res.RequestHeadersSnapshot) > 0 {
		if b, err := json.Marshal(res.RequestHeadersSnapshot); err == nil {
			s := string(b)
			reqHdrJSON = &s
		}
	}
	var respHdrJSON *string
	if len(res.ResponseHeaders) > 0 {
		if b, err := json.Marshal(res.ResponseHeaders); err == nil {
			s := string(b)
			respHdrJSON = &s
		}
	}

	var reqBody *string
	if res.RequestBodySnapshot != "" {
		s := res.RequestBodySnapshot
		reqBody = &s
	}

	respText := res.ResponseBody
	if strings.TrimSpace(respText) == "" && res.ErrorMessage != "" {
		respText = res.ErrorMessage
	}
	var respBody *string
	if respText != "" {
		respBody = &respText
	}

	dms := int(res.DurationMs)
	if dms < 0 {
		dms = 0
	}
	rsz := int(res.ResponseSizeBytes)
	if rsz < 0 {
		rsz = 0
	}

	item := &entity.HistoryItem{
		Method:              method,
		URL:                 urlStr,
		StatusCode:          res.StatusCode,
		DurationMs:          &dms,
		ResponseSizeBytes:   &rsz,
		RequestHeadersJSON:  reqHdrJSON,
		ResponseHeadersJSON: respHdrJSON,
		RequestBody:         reqBody,
		ResponseBody:        respBody,
		CreatedAt:           time.Now(),
	}
	if in != nil {
		item.WorkspaceID = trimmedStringPtr(in.WorkspaceID)
		item.RequestID = trimmedStringPtr(in.RequestID)
	}

	if err := h.history.Save(ctx, item); err != nil {
		logger.L().ErrorContext(ctx, "history save failed", "error", err)
	}
}

func trimmedStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}

// ImportFromCurl parses a shell-style cURL command into HTTPExecuteInput for the request editor.
func (h *HTTPHandler) ImportFromCurl(cmd string) (*entity.HTTPExecuteInput, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "HTTPHandler.ImportFromCurl")
	return service.ParseCurlCommand(cmd)
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
