package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/usecase"
	"context"
)

type SavedRequestHandler struct {
	ctx context.Context
	uc  usecase.RequestUsecase
}

func NewSavedRequestHandler(uc usecase.RequestUsecase) *SavedRequestHandler {
	return &SavedRequestHandler{uc: uc}
}

func (h *SavedRequestHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *SavedRequestHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

func (h *SavedRequestHandler) Create(in *entity.SavedRequestFull) (*entity.SavedRequestFull, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "SavedRequestHandler.Create", "folder_id", in.FolderID)
	return h.uc.CreateRequest(ctx, in)
}

func (h *SavedRequestHandler) Update(in *entity.SavedRequestFull) error {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "SavedRequestHandler.Update", "id", in.ID)
	return h.uc.UpdateRequest(ctx, in)
}

func (h *SavedRequestHandler) Delete(id string) error {
	ctx := h.getContext()
	return h.uc.DeleteRequest(ctx, id)
}

func (h *SavedRequestHandler) Get(id string) (*entity.SavedRequestFull, error) {
	ctx := h.getContext()
	return h.uc.GetRequest(ctx, id)
}

func (h *SavedRequestHandler) ListByFolder(folderID string) ([]*entity.SavedRequestSummary, error) {
	ctx := h.getContext()
	return h.uc.ListRequestsInFolder(ctx, folderID)
}
