package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/usecase"
	"context"
	"errors"
)

// FolderHandler replaces workspace + collection: nested folders.
type FolderHandler struct {
	ctx context.Context
	uc  usecase.FolderUsecase
}

func NewFolderHandler(uc usecase.FolderUsecase) *FolderHandler {
	return &FolderHandler{uc: uc}
}

func (h *FolderHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *FolderHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

func (h *FolderHandler) ListRootFolders() ([]*entity.FolderItem, error) {
	ctx := h.getContext()
	return h.uc.ListRootFolders(ctx)
}

func (h *FolderHandler) CreateFolder(in *entity.CreateFolderInput) (*entity.FolderItem, error) {
	ctx := h.getContext()
	if in == nil {
		return nil, errors.New("folder payload is nil")
	}
	logger.D().InfoContext(ctx, "FolderHandler.CreateFolder", "name", in.Name)
	item, appErr := h.uc.CreateFolder(ctx, in)
	if appErr != nil {
		return nil, appErr
	}
	return item, nil
}

func (h *FolderHandler) UpdateFolder(id, name, description string) error {
	ctx := h.getContext()
	return h.uc.UpdateFolder(ctx, id, name, description)
}

func (h *FolderHandler) DeleteFolder(id string) error {
	ctx := h.getContext()
	return h.uc.DeleteFolder(ctx, id)
}

func (h *FolderHandler) ListChildFolders(parentID string) ([]*entity.FolderItem, error) {
	ctx := h.getContext()
	return h.uc.ListChildFolders(ctx, parentID)
}

// MoveFolder re-parents a folder. `newParentID` empty string = move to root.
func (h *FolderHandler) MoveFolder(folderID, newParentID string) error {
	ctx := h.getContext()
	return h.uc.MoveFolder(ctx, folderID, newParentID)
}

// ReorderFolder moves a folder among siblings under parentID (empty = roots).
// insertBeforeID empty = append at end; otherwise insert before that sibling id.
func (h *FolderHandler) ReorderFolder(folderID, parentID, insertBeforeID string) error {
	ctx := h.getContext()
	return h.uc.ReorderFolder(ctx, folderID, parentID, insertBeforeID)
}
