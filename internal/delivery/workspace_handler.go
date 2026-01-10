package delivery

import (
	"PostmanJanai/internal/entity"
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

func (h *WorkspaceHandler) CreateWorkspace(item *entity.WorkspaceItem) (*entity.WorkspaceItem, error) {
	return h.workspaceUC.CreateWorkspace(h.ctx, item)
}

func (h *WorkspaceHandler) GetAll() ([]*entity.WorkspaceItem, error) {
	return h.workspaceUC.ListWorkspaces(h.ctx)
}

func (h *WorkspaceHandler) Update(id int, name, desc string) (string, error) {
	err := h.workspaceUC.UpdateWorkspace(h.ctx, id, name, desc)
	if err != nil {
		return "Lỗi", err
	}
	return "Cập nhật thành công", nil
}

func (h *WorkspaceHandler) Delete(id int) error {
	return h.workspaceUC.DeleteWorkspace(h.ctx, id)
}
