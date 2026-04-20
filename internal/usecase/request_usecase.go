package usecase

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/repository"
	"context"
	"strings"

	"github.com/google/uuid"
)

type RequestUsecase interface {
	CreateRequest(ctx context.Context, in *entity.SavedRequestFull) (*entity.SavedRequestFull, error)
	UpdateRequest(ctx context.Context, in *entity.SavedRequestFull) error
	DeleteRequest(ctx context.Context, id string) error
	GetRequest(ctx context.Context, id string) (*entity.SavedRequestFull, error)
	ListRequestsInFolder(ctx context.Context, folderID string) ([]*entity.SavedRequestSummary, error)
	MoveRequest(ctx context.Context, requestID, folderID string) error
}

type requestUsecaseImpl struct {
	folders repository.FolderRepository
	savedR  repository.RequestRepository
}

func NewRequestUsecase(folders repository.FolderRepository, savedR repository.RequestRepository) RequestUsecase {
	return &requestUsecaseImpl{folders: folders, savedR: savedR}
}

func (u *requestUsecaseImpl) validateFolder(ctx context.Context, folderID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(folderID)); err != nil {
		return err
	}
	_, err := u.folders.GetByID(ctx, folderID)
	return err
}

func (u *requestUsecaseImpl) checkName(ctx context.Context, in *entity.SavedRequestFull, excludeID *string) error {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	taken, err := u.savedR.ExistsNameInFolder(ctx, in.FolderID, name, excludeID)
	if err != nil {
		return err
	}
	if taken {
		return apperror.NewWithErrorDetail(constant.ErrSavedRequestNameConflict, nil)
	}
	return nil
}

func (u *requestUsecaseImpl) CreateRequest(ctx context.Context, in *entity.SavedRequestFull) (*entity.SavedRequestFull, error) {
	if in == nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	if err := u.validateFolder(ctx, in.FolderID); err != nil {
		return nil, err
	}
	if err := u.checkName(ctx, in, nil); err != nil {
		return nil, err
	}
	normalizeRequestPayload(in)
	id, err := u.savedR.CreateFull(ctx, in)
	if err != nil {
		return nil, err
	}
	return u.savedR.GetByID(ctx, id)
}

func (u *requestUsecaseImpl) UpdateRequest(ctx context.Context, in *entity.SavedRequestFull) error {
	if in == nil || strings.TrimSpace(in.ID) == "" {
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	if _, err := u.savedR.GetByID(ctx, in.ID); err != nil {
		return err
	}
	if err := u.validateFolder(ctx, in.FolderID); err != nil {
		return err
	}
	if err := u.checkName(ctx, in, &in.ID); err != nil {
		return err
	}
	normalizeRequestPayload(in)
	return u.savedR.UpdateFull(ctx, in)
}

func normalizeRequestPayload(in *entity.SavedRequestFull) {
	in.Method = strings.TrimSpace(in.Method)
	if in.Method == "" {
		in.Method = "GET"
	}
	in.URL = strings.TrimSpace(in.URL)
	if in.URL == "" {
		in.URL = "https://"
	}
	in.BodyMode = strings.TrimSpace(in.BodyMode)
	if in.BodyMode == "" {
		in.BodyMode = "none"
	}
	in.Name = strings.TrimSpace(in.Name)
}

func (u *requestUsecaseImpl) DeleteRequest(ctx context.Context, id string) error {
	if _, err := u.savedR.GetByID(ctx, id); err != nil {
		return err
	}
	return u.savedR.DeleteByID(ctx, id)
}

func (u *requestUsecaseImpl) GetRequest(ctx context.Context, id string) (*entity.SavedRequestFull, error) {
	return u.savedR.GetByID(ctx, id)
}

func (u *requestUsecaseImpl) ListRequestsInFolder(ctx context.Context, folderID string) ([]*entity.SavedRequestSummary, error) {
	if _, err := u.folders.GetByID(ctx, folderID); err != nil {
		return nil, err
	}
	return u.savedR.ListByFolder(ctx, folderID)
}

func (u *requestUsecaseImpl) MoveRequest(ctx context.Context, requestID, folderID string) error {
	requestID = strings.TrimSpace(requestID)
	folderID = strings.TrimSpace(folderID)
	if requestID == "" || folderID == "" {
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	full, err := u.savedR.GetByID(ctx, requestID)
	if err != nil {
		return err
	}
	if full.FolderID == folderID {
		return nil
	}
	if err := u.validateFolder(ctx, folderID); err != nil {
		return err
	}
	dup := *full
	dup.FolderID = folderID
	if err := u.checkName(ctx, &dup, &requestID); err != nil {
		return err
	}
	return u.savedR.MoveToFolder(ctx, requestID, folderID)
}
