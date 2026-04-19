package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/usecase"
	"context"
)

// EnvironmentHandler exposes environment CRUD to the Wails frontend.
type EnvironmentHandler struct {
	ctx context.Context
	uc  usecase.EnvironmentUsecase
}

func NewEnvironmentHandler(uc usecase.EnvironmentUsecase) *EnvironmentHandler {
	return &EnvironmentHandler{uc: uc}
}

func (h *EnvironmentHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *EnvironmentHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

func (h *EnvironmentHandler) List() ([]entity.EnvironmentSummary, error) {
	ctx := h.getContext()
	return h.uc.List(ctx)
}

func (h *EnvironmentHandler) Get(id string) (*entity.EnvironmentFull, error) {
	ctx := h.getContext()
	return h.uc.Get(ctx, id)
}

func (h *EnvironmentHandler) Create(name, description string) (*entity.EnvironmentFull, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "EnvironmentHandler.Create", "name", name)
	return h.uc.Create(ctx, name, description)
}

func (h *EnvironmentHandler) UpdateMeta(id, name, description string) error {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "EnvironmentHandler.UpdateMeta", "id", id)
	return h.uc.UpdateMeta(ctx, id, name, description)
}

func (h *EnvironmentHandler) Delete(id string) error {
	ctx := h.getContext()
	return h.uc.Delete(ctx, id)
}

func (h *EnvironmentHandler) SaveVariables(envID string, variables []entity.EnvVariableInput) error {
	ctx := h.getContext()
	return h.uc.SaveVariables(ctx, envID, variables)
}

func (h *EnvironmentHandler) SetActive(id string) error {
	ctx := h.getContext()
	return h.uc.SetActive(ctx, id)
}

func (h *EnvironmentHandler) ClearActive() error {
	ctx := h.getContext()
	return h.uc.ClearActive(ctx)
}

func (h *EnvironmentHandler) GetActive() (*entity.EnvironmentSummary, error) {
	ctx := h.getContext()
	return h.uc.GetActiveSummary(ctx)
}
