package delivery

import (
	"PostmanJanai/internal/constant"
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

func savedRequestID(in *entity.HTTPExecuteInput) string {
	if in == nil || in.RequestID == nil {
		return ""
	}
	return strings.TrimSpace(*in.RequestID)
}

// applyCaptureAndAssertions runs Phase 8 capture + assertion rules attached to the
// saved request (when one is identified) against the response and writes capture
// outcomes into the active environment for environment-scoped rules. Failures
// degrade gracefully — they're surfaced via the returned result, never propagated.
func (h *HTTPHandler) applyCaptureAndAssertions(ctx context.Context, requestID string, res *entity.HTTPExecuteResult) {
	if h.rules == nil || res == nil || strings.TrimSpace(requestID) == "" {
		return
	}
	if res.ErrorMessage != "" {
		return
	}
	captureRules, cerr := h.rules.ListCaptures(ctx, requestID)
	if cerr != nil {
		logger.L().InfoContext(ctx, "list captures failed", "error", cerr)
	}
	assertionRules, aerr := h.rules.ListAssertions(ctx, requestID)
	if aerr != nil {
		logger.L().InfoContext(ctx, "list assertions failed", "error", aerr)
	}
	if len(captureRules) == 0 && len(assertionRules) == 0 {
		return
	}
	capCtx := service.NewCaptureContext(res.StatusCode, res.ResponseHeaders, res.ResponseBody)
	if len(captureRules) > 0 {
		captures := service.RunCaptureRules(capCtx, captureRules)
		for i := range captures {
			c := &captures[i]
			if !c.Captured {
				continue
			}
			scope := strings.TrimSpace(c.TargetScope)
			if scope == "" {
				scope = constant.CaptureScopeEnvironment
			}
			if scope != constant.CaptureScopeEnvironment {
				continue
			}
			if h.env == nil {
				c.ErrorMessage = "no environment repository configured"
				continue
			}
			ok, err := h.env.UpsertActiveVariable(ctx, c.TargetVariable, c.Value)
			if err != nil {
				c.ErrorMessage = err.Error()
				continue
			}
			if !ok {
				c.ErrorMessage = "no active environment to receive capture"
			}
		}
		res.Captures = captures
	}
	if len(assertionRules) > 0 {
		assertCtx := service.AssertionContextFromCapture(capCtx, res.DurationMs, res.ResponseSizeBytes)
		res.Assertions = service.RunAssertionRules(assertCtx, assertionRules)
	}
}

// HTTPHandler is the Wails binding for HTTP request execution (Phase 1).
type HTTPHandler struct {
	ctx      context.Context
	executor *service.HTTPExecutor
	history  repository.HistoryRepository
	env      repository.EnvironmentRepository
	saved    repository.RequestRepository
	rules    repository.RequestRuleRepository
}

func NewHTTPHandler(
	ex *service.HTTPExecutor,
	hist repository.HistoryRepository,
	env repository.EnvironmentRepository,
	saved repository.RequestRepository,
	rules repository.RequestRuleRepository,
) *HTTPHandler {
	return &HTTPHandler{executor: ex, history: hist, env: env, saved: saved, rules: rules}
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
	vars := map[string]string{}
	if h.env != nil {
		m, err := h.env.ActiveVariableMap(ctx)
		if err != nil {
			logger.L().InfoContext(ctx, "active environment variables unavailable", "error", err)
		} else if m != nil {
			vars = m
		}
	}
	resolved := service.CloneSubstituteHTTPExecuteInput(in, vars)
	service.MergeAuthIntoHeadersAndQuery(resolved)

	if resolved != nil && h.saved != nil && resolved.RequestID != nil && strings.TrimSpace(*resolved.RequestID) != "" {
		if full, err := h.saved.GetByID(ctx, strings.TrimSpace(*resolved.RequestID)); err == nil && full != nil {
			resolved.InsecureSkipVerify = resolved.InsecureSkipVerify || full.InsecureSkipVerify
		}
	}

	res, err := h.executor.Execute(ctx, resolved)
	if err != nil {
		logger.L().ErrorContext(ctx, "HTTP execute validation failed", "error", err)
		return nil, err
	}
	if res.ErrorMessage != "" {
		logger.L().InfoContext(ctx, "HTTP execute finished with transport error", "error", res.ErrorMessage)
	} else {
		logger.L().InfoContext(ctx, "HTTP execute success", "status", res.StatusCode, "duration_ms", res.DurationMs)
	}

	if requestID := savedRequestID(in); requestID != "" {
		h.applyCaptureAndAssertions(ctx, requestID, res)
	}
	secrets := []string{}
	if h.env != nil {
		if s, err := h.env.ActiveSecretPlaintexts(ctx); err == nil && s != nil {
			secrets = s
		}
	}
	histIn := resolved
	if len(secrets) > 0 {
		histIn = service.RedactHTTPExecuteInput(resolved, secrets)
	}

	insecureTLS := false
	if resolved != nil {
		insecureTLS = resolved.InsecureSkipVerify
	}

	// History stores the request as actually sent (resolved), not template placeholders.
	h.persistHistory(ctx, histIn, res, insecureTLS)
	return res, nil
}

func (h *HTTPHandler) persistHistory(ctx context.Context, in *entity.HTTPExecuteInput, res *entity.HTTPExecuteResult, insecureTLS bool) {
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
	reqHdrSnap := res.RequestHeadersSnapshot
	reqBodySnap := res.RequestBodySnapshot
	if in != nil {
		if fu, hdrs, body, err := service.HTTPRequestSnapshotsForHistory(ctx, in); err == nil {
			urlStr = strings.TrimSpace(fu)
			reqHdrSnap = hdrs
			reqBodySnap = body
		}
	}
	if urlStr == "" && in != nil {
		urlStr = strings.TrimSpace(in.URL)
	}

	var reqHdrJSON *string
	if len(reqHdrSnap) > 0 {
		if b, err := json.Marshal(reqHdrSnap); err == nil {
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
	if reqBodySnap != "" {
		s := reqBodySnap
		reqBody = &s
	}

	respText := res.ResponseBody
	if strings.TrimSpace(respText) == "" && res.ErrorMessage != "" {
		respText = res.ErrorMessage
	}
	if res.BodyTruncated && strings.TrimSpace(respText) != "" {
		respText += "\n\n[… response body truncated at configured max size …]"
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
		InsecureTLS:         insecureTLS,
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
		item.RootFolderID = trimmedStringPtr(in.RootFolderID)
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
