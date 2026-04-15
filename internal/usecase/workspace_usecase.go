package usecase

import (
	"PostmanJanai/ent"
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"context"
	"errors"
	"strings"
)

type WorkspaceUsecase interface {
	CreateWorkspace(ctx context.Context, item *entity.WorkspaceItem) (*entity.WorkspaceItem, *apperror.AppError)
	ListWorkspaces(ctx context.Context) ([]*entity.WorkspaceItem, error)
	UpdateWorkspace(ctx context.Context, id int, name, desc string) error
	DeleteWorkspace(ctx context.Context, id int) error
}

type workspaceUsecaseImpl struct {
	workspaceRepo repository.WorkspaceRepository
}

func NewWorkspaceUsecase(workspaceRepo repository.WorkspaceRepository) WorkspaceUsecase {
	return &workspaceUsecaseImpl{
		workspaceRepo: workspaceRepo,
	}
}

func (w *workspaceUsecaseImpl) CreateWorkspace(ctx context.Context, item *entity.WorkspaceItem) (*entity.WorkspaceItem, *apperror.AppError) {
	if item == nil {
		err := apperror.NewWithErrorDetail(constant.ErrInternal, errors.New("workspace payload is nil"))
		logger.L().ErrorContext(ctx, "CreateWorkspace failed: nil payload", "error", err)
		return nil, err
	}
	item.WorkspaceName = strings.TrimSpace(item.WorkspaceName)
	item.WorkspaceDescription = strings.TrimSpace(item.WorkspaceDescription)
	if item.WorkspaceName == "" {
		err := apperror.NewWithErrorDetail(constant.ErrInternal, errors.New("workspace name is empty"))
		logger.L().ErrorContext(ctx, "CreateWorkspace failed: empty workspace name", "error", err)
		return nil, err
	}

	logger.D().InfoContext(ctx, "CreateWorkspace request received",
		"workspace_name", item.WorkspaceName,
		"description_length", len(item.WorkspaceDescription),
	)
	wsDb, err := w.workspaceRepo.GetByName(ctx, item.WorkspaceName)
	if err != nil && !ent.IsNotFound(err) {
		logger.L().ErrorContext(ctx, "CreateWorkspace - get workspace by name error", "name", item.WorkspaceName, "error", err)
		return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	if wsDb != nil {
		logger.L().WarnContext(ctx, "CreateWorkspace blocked: workspace already exists", "name", item.WorkspaceName)
		return nil, apperror.NewWithErrorDetail(constant.ErrWorkspaceAlreadyExisted, nil)
	}

	logger.D().InfoContext(ctx, "CreateWorkspace name is available", "workspace_name", item.WorkspaceName)
	wsID, err := w.workspaceRepo.Save(ctx, item)
	if err != nil {
		logger.L().ErrorContext(ctx, "CreateWorkspace - save workspace error", "error", err)
		return nil, apperror.NewWithErrorDetail(constant.ErrWorkspaceSaveFail, err)
	}
	item.ID = wsID
	logger.D().InfoContext(ctx, "CreateWorkspace success", "id", item.ID, "workspace_name", item.WorkspaceName)
	return item, nil
}

func (uc *workspaceUsecaseImpl) ListWorkspaces(ctx context.Context) ([]*entity.WorkspaceItem, error) {
	items, err := uc.workspaceRepo.GetAll(ctx)
	if err != nil {
		logger.L().ErrorContext(ctx, "ListWorkspaces failed", "error", err)
		return nil, err
	}
	logger.D().InfoContext(ctx, "ListWorkspaces success", "count", len(items))
	return items, nil
}

func (uc *workspaceUsecaseImpl) UpdateWorkspace(ctx context.Context, id int, name, desc string) error {
	name = strings.TrimSpace(name)
	desc = strings.TrimSpace(desc)
	if name == "" {
		err := errors.New("workspace name is empty")
		logger.L().ErrorContext(ctx, "UpdateWorkspace failed: empty name", "id", id, "error", err)
		return err
	}
	err := uc.workspaceRepo.UpdateByID(ctx, &entity.WorkspaceItem{ID: id, WorkspaceName: name, WorkspaceDescription: desc})
	if err != nil {
		logger.L().ErrorContext(ctx, "UpdateWorkspace - update workspace error", "error", err)
		return err
	}
	logger.D().InfoContext(ctx, "UpdateWorkspace success", "id", id, "workspace_name", name)
	return nil
}

func (uc *workspaceUsecaseImpl) DeleteWorkspace(ctx context.Context, id int) error {
	if err := uc.workspaceRepo.DeleteByID(ctx, id); err != nil {
		logger.L().ErrorContext(ctx, "DeleteWorkspace failed", "id", id, "error", err)
		return err
	}
	logger.D().InfoContext(ctx, "DeleteWorkspace success", "id", id)
	return nil
}
