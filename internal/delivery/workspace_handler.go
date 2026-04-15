package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/usecase"
	"context"
)

type WorkspaceHandler struct {
	ctx         context.Context
	workspaceUC usecase.WorkspaceUsecase
}

func NewWorkspaceHandler(uc usecase.WorkspaceUsecase) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceUC: uc}
}

func (h *WorkspaceHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *WorkspaceHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

func (h *WorkspaceHandler) CreateWorkspace(item *entity.WorkspaceItem) (*entity.WorkspaceItem, error) {
	ctx := h.getContext()
	logger.L().InfoContext(ctx, "Workspace action started", "action", "create")
	logger.D().InfoContext(ctx, "WorkspaceHandler.CreateWorkspace called", "payload", item)
	res, err := h.workspaceUC.CreateWorkspace(ctx, item)
	if err != nil {
		logger.L().ErrorContext(ctx, "Workspace action failed", "action", "create", "error", err)
		logger.D().ErrorContext(ctx, "WorkspaceHandler.CreateWorkspace failed", "error", err)
		return nil, err
	}
	logger.L().InfoContext(ctx, "Workspace action success", "action", "create", "id", res.ID)
	logger.D().InfoContext(ctx, "WorkspaceHandler.CreateWorkspace success", "id", res.ID)
	return res, nil
}

func (h *WorkspaceHandler) GetAll() ([]*entity.WorkspaceItem, error) {
	ctx := h.getContext()
	logger.L().InfoContext(ctx, "Workspace action started", "action", "list")
	logger.D().InfoContext(ctx, "WorkspaceHandler.GetAll called")
	items, err := h.workspaceUC.ListWorkspaces(ctx)
	if err != nil {
		logger.L().ErrorContext(ctx, "Workspace action failed", "action", "list", "error", err)
		logger.D().ErrorContext(ctx, "WorkspaceHandler.GetAll failed", "error", err)
		return nil, err
	}
	logger.L().InfoContext(ctx, "Workspace action success", "action", "list", "count", len(items))
	logger.D().InfoContext(ctx, "WorkspaceHandler.GetAll success", "count", len(items))
	return items, nil
}

func (h *WorkspaceHandler) Update(id string, name, desc string) (string, error) {
	ctx := h.getContext()
	logger.L().InfoContext(ctx, "Workspace action started", "action", "update", "id", id)
	logger.D().InfoContext(ctx, "WorkspaceHandler.Update called", "id", id, "name", name)
	err := h.workspaceUC.UpdateWorkspace(ctx, id, name, desc)
	if err != nil {
		logger.L().ErrorContext(ctx, "Workspace action failed", "action", "update", "id", id, "error", err)
		logger.D().ErrorContext(ctx, "WorkspaceHandler.Update failed", "id", id, "error", err)
		return "Lỗi", err
	}
	logger.L().InfoContext(ctx, "Workspace action success", "action", "update", "id", id)
	return "Cập nhật thành công", nil
}

func (h *WorkspaceHandler) Delete(id string) error {
	ctx := h.getContext()
	logger.L().InfoContext(ctx, "Workspace action started", "action", "delete", "id", id)
	logger.D().InfoContext(ctx, "WorkspaceHandler.Delete called", "id", id)
	err := h.workspaceUC.DeleteWorkspace(ctx, id)
	if err != nil {
		logger.L().ErrorContext(ctx, "Workspace action failed", "action", "delete", "id", id, "error", err)
		logger.D().ErrorContext(ctx, "WorkspaceHandler.Delete failed", "id", id, "error", err)
		return err
	}
	logger.L().InfoContext(ctx, "Workspace action success", "action", "delete", "id", id)
	return nil
}
