package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"context"
	"errors"
	"strings"
)

// RuleHandler exposes Phase 8 capture + assertion CRUD over Wails.
//
// Both endpoints accept the saved request UUID and the full intended list, so
// the frontend rule editor stays simple (always submits the entire list, never
// per-row deltas).
type RuleHandler struct {
	ctx   context.Context
	rules repository.RequestRuleRepository
}

func NewRuleHandler(rules repository.RequestRuleRepository) *RuleHandler {
	return &RuleHandler{rules: rules}
}

func (h *RuleHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *RuleHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

func (h *RuleHandler) ListCaptures(requestID string) ([]entity.RequestCaptureRow, error) {
	ctx := h.getContext()
	if strings.TrimSpace(requestID) == "" {
		return nil, errors.New("request id is required")
	}
	logger.D().InfoContext(ctx, "RuleHandler.ListCaptures", "request_id", requestID)
	return h.rules.ListCaptures(ctx, requestID)
}

func (h *RuleHandler) SaveCaptures(requestID string, rows []entity.RequestCaptureInput) ([]entity.RequestCaptureRow, error) {
	ctx := h.getContext()
	if strings.TrimSpace(requestID) == "" {
		return nil, errors.New("request id is required")
	}
	logger.D().InfoContext(ctx, "RuleHandler.SaveCaptures", "request_id", requestID, "count", len(rows))
	return h.rules.SaveCaptures(ctx, requestID, rows)
}

func (h *RuleHandler) ListAssertions(requestID string) ([]entity.RequestAssertionRow, error) {
	ctx := h.getContext()
	if strings.TrimSpace(requestID) == "" {
		return nil, errors.New("request id is required")
	}
	logger.D().InfoContext(ctx, "RuleHandler.ListAssertions", "request_id", requestID)
	return h.rules.ListAssertions(ctx, requestID)
}

func (h *RuleHandler) SaveAssertions(requestID string, rows []entity.RequestAssertionInput) ([]entity.RequestAssertionRow, error) {
	ctx := h.getContext()
	if strings.TrimSpace(requestID) == "" {
		return nil, errors.New("request id is required")
	}
	logger.D().InfoContext(ctx, "RuleHandler.SaveAssertions", "request_id", requestID, "count", len(rows))
	return h.rules.SaveAssertions(ctx, requestID, rows)
}
