package usecase

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"context"
	"database/sql"
	"errors"
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
	logger.L().InfoContext(ctx, "Start CreateWorkspace, item: %v", item)
	wsDb, err := w.workspaceRepo.GetByName(ctx, item.WorkspaceName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.L().ErrorContext(ctx, "CreateWorkspace - get workspace by name error, name: %v, err: %v", item.WorkspaceName, err)
		return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	if wsDb != nil {
		logger.L().ErrorContext(ctx, "CreateWorkspace - workspace already exists, name: %v, item: %v", item.WorkspaceName, item)
		return nil, apperror.NewWithErrorDetail(constant.ErrWorkspaceAlreadyExisted, nil)
	}

	wsID, err := w.workspaceRepo.Save(ctx, item)
	if err != nil {
		logger.L().ErrorContext(ctx, "CreateWorkspace - save workspace error, err: %v", err)
		return nil, apperror.NewWithErrorDetail(constant.ErrWorkspaceSaveFail, err)
	}
	item.ID = wsID
	return item, nil
}

func (uc *workspaceUsecaseImpl) ListWorkspaces(ctx context.Context) ([]*entity.WorkspaceItem, error) {
	return uc.workspaceRepo.GetAll(ctx)
}

func (uc *workspaceUsecaseImpl) UpdateWorkspace(ctx context.Context, id int, name, desc string) error {
	_, err := uc.workspaceRepo.Save(ctx, &entity.WorkspaceItem{ID: id, WorkspaceName: name, WorkspaceDescription: desc})
	if err != nil {
		logger.L().ErrorContext(ctx, "UpdateWorkspace - update workspace error, err: %v", err)
	}
	return err
}

func (uc *workspaceUsecaseImpl) DeleteWorkspace(ctx context.Context, id int) error {
	return uc.workspaceRepo.DeleteByID(ctx, id)
}
