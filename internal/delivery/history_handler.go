package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"context"
)

// HistoryHandler exposes request run history to the UI (Phase 1).
type HistoryHandler struct {
	ctx     context.Context
	history repository.HistoryRepository
}

func NewHistoryHandler(hist repository.HistoryRepository) *HistoryHandler {
	return &HistoryHandler{history: hist}
}

func (h *HistoryHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *HistoryHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

// List returns recent history rows, newest first (see repository ordering).
func (h *HistoryHandler) List() ([]entity.HistoryItem, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "HistoryHandler.List called")
	items, err := h.history.GetAll(ctx)
	if err != nil {
		logger.L().ErrorContext(ctx, "HistoryHandler.List failed", "error", err)
		return nil, err
	}
	logger.D().InfoContext(ctx, "HistoryHandler.List success", "count", len(items))
	return items, nil
}
