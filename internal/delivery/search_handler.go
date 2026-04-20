package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/usecase"
	"context"
)

// SearchHandler exposes sidebar search (folders + saved requests) via Wails.
// History search/filter is client-side and has no counterpart here.
type SearchHandler struct {
	ctx context.Context
	uc  usecase.SearchUsecase
}

func NewSearchHandler(uc usecase.SearchUsecase) *SearchHandler {
	return &SearchHandler{uc: uc}
}

func (h *SearchHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *SearchHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

// SearchTree matches folders (by name) and saved requests (by name or URL),
// case-insensitive substring. Empty query returns empty results, which the
// sidebar uses to revert to the regular tree view.
func (h *SearchHandler) SearchTree(query string, limit int) (*entity.SearchResults, error) {
	return h.uc.SearchTree(h.getContext(), query, limit)
}
