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

	"github.com/google/uuid"
)

type FolderUsecase interface {
	CreateFolder(ctx context.Context, in *entity.CreateFolderInput) (*entity.FolderItem, *apperror.AppError)
	ListRootFolders(ctx context.Context) ([]*entity.FolderItem, error)
	ListChildFolders(ctx context.Context, parentID string) ([]*entity.FolderItem, error)
	UpdateFolder(ctx context.Context, id, name, desc string) error
	DeleteFolder(ctx context.Context, id string) error
}

type folderUsecaseImpl struct {
	folders repository.FolderRepository
}

func NewFolderUsecase(folders repository.FolderRepository) FolderUsecase {
	return &folderUsecaseImpl{folders: folders}
}

func (u *folderUsecaseImpl) CreateFolder(ctx context.Context, in *entity.CreateFolderInput) (*entity.FolderItem, *apperror.AppError) {
	if in == nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInternal, errors.New("nil folder payload"))
	}
	name := strings.TrimSpace(in.Name)
	desc := strings.TrimSpace(in.Description)
	if name == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInternal, errors.New("empty folder name"))
	}
	if in.ParentID == nil || strings.TrimSpace(*in.ParentID) == "" {
		taken, err := u.folders.RootNameTaken(ctx, name, nil)
		if err != nil {
			return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
		}
		if taken {
			return nil, apperror.NewWithErrorDetail(constant.ErrFolderRootNameConflict, nil)
		}
	} else {
		pid := strings.TrimSpace(*in.ParentID)
		if _, err := uuid.Parse(pid); err != nil {
			return nil, apperror.NewWithErrorDetail(constant.ErrInternal, err)
		}
		if _, err := u.folders.GetByID(ctx, pid); err != nil {
			if ent.IsNotFound(err) {
				return nil, apperror.NewWithErrorDetail(constant.ErrFolderNotFound, nil)
			}
			return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
		}
		taken, err := u.folders.ChildNameTaken(ctx, pid, name, nil)
		if err != nil || taken {
			if taken {
				return nil, apperror.NewWithErrorDetail(constant.ErrFolderChildNameConflict, nil)
			}
			return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
		}
	}
	id, err := u.folders.Create(ctx, &entity.FolderItem{
		ParentID:    in.ParentID,
		Name:        name,
		Description: desc,
	})
	if err != nil {
		logger.L().ErrorContext(ctx, "CreateFolder save failed", "error", err)
		return nil, apperror.NewWithErrorDetail(constant.ErrFolderSaveFail, err)
	}
	item, err := u.folders.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	return item, nil
}

func (u *folderUsecaseImpl) ListRootFolders(ctx context.Context) ([]*entity.FolderItem, error) {
	return u.folders.ListRoots(ctx)
}

func (u *folderUsecaseImpl) ListChildFolders(ctx context.Context, parentID string) ([]*entity.FolderItem, error) {
	if _, err := uuid.Parse(strings.TrimSpace(parentID)); err != nil {
		return nil, err
	}
	if _, err := u.folders.GetByID(ctx, parentID); err != nil {
		return nil, err
	}
	return u.folders.ListChildren(ctx, parentID)
}

func (u *folderUsecaseImpl) UpdateFolder(ctx context.Context, id, name, desc string) error {
	name = strings.TrimSpace(name)
	desc = strings.TrimSpace(desc)
	if name == "" {
		return errors.New("empty folder name")
	}
	existing, err := u.folders.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ParentID == nil {
		taken, err := u.folders.RootNameTaken(ctx, name, &id)
		if err != nil {
			return err
		}
		if taken {
			return apperror.NewWithErrorDetail(constant.ErrFolderRootNameConflict, nil)
		}
	} else {
		taken, err := u.folders.ChildNameTaken(ctx, *existing.ParentID, name, &id)
		if err != nil {
			return err
		}
		if taken {
			return apperror.NewWithErrorDetail(constant.ErrFolderChildNameConflict, nil)
		}
	}
	return u.folders.UpdateByID(ctx, &entity.FolderItem{
		ID:          id,
		ParentID:    existing.ParentID,
		Name:        name,
		Description: desc,
	})
}

func (u *folderUsecaseImpl) DeleteFolder(ctx context.Context, id string) error {
	if _, err := u.folders.GetByID(ctx, id); err != nil {
		return err
	}
	return u.folders.DeleteByID(ctx, id)
}
