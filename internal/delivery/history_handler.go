package delivery

import (
	"PostmanJanai/ent"
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"context"
	"strings"
)

// HistoryHandler exposes request run history to the UI (Phase 1 + Phase 3 detail).
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

// List returns history rows for the sidebar (newest first). Pass empty rootFolderID to include all roots.
func (h *HistoryHandler) List(rootFolderID string) ([]entity.HistorySummary, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "HistoryHandler.List called", "root_folder_id", rootFolderID)
	var filter *string
	if s := strings.TrimSpace(rootFolderID); s != "" {
		filter = &s
	}
	items, err := h.history.ListSummaries(ctx, filter)
	if err != nil {
		logger.L().ErrorContext(ctx, "HistoryHandler.List failed", "error", err)
		return nil, err
	}
	logger.D().InfoContext(ctx, "HistoryHandler.List success", "count", len(items))
	return items, nil
}

// Get returns one history row with full request/response snapshot (headers + bodies).
func (h *HistoryHandler) Get(id string) (*entity.HistoryItem, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "HistoryHandler.Get", "id", id)
	item, err := h.history.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, apperror.NewWithErrorDetail(constant.ErrHistoryNotFound, err)
		}
		logger.L().ErrorContext(ctx, "HistoryHandler.Get failed", "error", err)
		return nil, err
	}
	return item, nil
}

// Delete removes one history row by id.
func (h *HistoryHandler) Delete(id string) error {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "HistoryHandler.Delete", "id", id)
	if err := h.history.DeleteByID(ctx, strings.TrimSpace(id)); err != nil {
		if ent.IsNotFound(err) {
			return apperror.NewWithErrorDetail(constant.ErrHistoryNotFound, err)
		}
		logger.L().ErrorContext(ctx, "HistoryHandler.Delete failed", "error", err)
		return err
	}
	return nil
}
