package usecase

import (
	"PostmanJanai/ent"
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/repository"
	"context"
	"strings"

	"github.com/google/uuid"
)

// EnvironmentUsecase handles environment CRUD and activation.
type EnvironmentUsecase interface {
	List(ctx context.Context) ([]entity.EnvironmentSummary, error)
	Get(ctx context.Context, id string) (*entity.EnvironmentFull, error)
	Create(ctx context.Context, name, description string) (*entity.EnvironmentFull, error)
	UpdateMeta(ctx context.Context, id, name, description string) error
	Delete(ctx context.Context, id string) error
	SaveVariables(ctx context.Context, envID string, rows []entity.EnvVariableInput) error
	SetActive(ctx context.Context, id string) error
	ClearActive(ctx context.Context) error
	GetActiveSummary(ctx context.Context) (*entity.EnvironmentSummary, error)
}

type environmentUsecaseImpl struct {
	repo repository.EnvironmentRepository
}

func NewEnvironmentUsecase(repo repository.EnvironmentRepository) EnvironmentUsecase {
	return &environmentUsecaseImpl{repo: repo}
}

func (u *environmentUsecaseImpl) List(ctx context.Context) ([]entity.EnvironmentSummary, error) {
	return u.repo.ListSummaries(ctx)
}

func (u *environmentUsecaseImpl) Get(ctx context.Context, id string) (*entity.EnvironmentFull, error) {
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return nil, err
	}
	full, err := u.repo.GetFull(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, apperror.NewWithErrorDetail(constant.ErrEnvironmentNotFound, nil)
		}
		return nil, err
	}
	return full, nil
}

func (u *environmentUsecaseImpl) Create(ctx context.Context, name, description string) (*entity.EnvironmentFull, error) {
	if strings.TrimSpace(name) == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	taken, err := u.repo.NameTaken(ctx, name, nil)
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	if taken {
		return nil, apperror.NewWithErrorDetail(constant.ErrEnvironmentNameConflict, nil)
	}
	full, err := u.repo.Create(ctx, name, description)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, apperror.NewWithErrorDetail(constant.ErrEnvironmentNameConflict, err)
		}
		return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	return full, nil
}

func (u *environmentUsecaseImpl) UpdateMeta(ctx context.Context, id, name, description string) error {
	if strings.TrimSpace(name) == "" {
		return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
	}
	if _, err := u.Get(ctx, id); err != nil {
		return err
	}
	taken, err := u.repo.NameTaken(ctx, name, &id)
	if err != nil {
		return apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	if taken {
		return apperror.NewWithErrorDetail(constant.ErrEnvironmentNameConflict, nil)
	}
	err = u.repo.UpdateMeta(ctx, id, name, description)
	if err != nil {
		if ent.IsNotFound(err) {
			return apperror.NewWithErrorDetail(constant.ErrEnvironmentNotFound, nil)
		}
		if ent.IsConstraintError(err) {
			return apperror.NewWithErrorDetail(constant.ErrEnvironmentNameConflict, err)
		}
		return apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	return nil
}

func (u *environmentUsecaseImpl) Delete(ctx context.Context, id string) error {
	if _, err := u.Get(ctx, id); err != nil {
		return err
	}
	return u.repo.Delete(ctx, id)
}

func (u *environmentUsecaseImpl) SaveVariables(ctx context.Context, envID string, rows []entity.EnvVariableInput) error {
	if _, err := u.Get(ctx, envID); err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, row := range rows {
		k := strings.TrimSpace(row.Key)
		if k == "" {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(row.Kind))
		if kind == "" {
			kind = constant.EnvVarKindPlain
		}
		if kind != constant.EnvVarKindPlain && kind != constant.EnvVarKindSecret {
			return apperror.NewWithErrorDetail(constant.ErrInternal, nil)
		}
		kl := strings.ToLower(k)
		if _, ok := seen[kl]; ok {
			return apperror.NewWithErrorDetail(constant.ErrEnvironmentDuplicateVariableKey, nil)
		}
		seen[kl] = struct{}{}
	}
	err := u.repo.SaveVariables(ctx, envID, rows)
	if err != nil {
		if ent.IsConstraintError(err) {
			return apperror.NewWithErrorDetail(constant.ErrInternal, err)
		}
		return err
	}
	return nil
}

func (u *environmentUsecaseImpl) SetActive(ctx context.Context, id string) error {
	if _, err := u.Get(ctx, id); err != nil {
		return err
	}
	return u.repo.SetActive(ctx, id)
}

func (u *environmentUsecaseImpl) ClearActive(ctx context.Context) error {
	return u.repo.ClearActive(ctx)
}

func (u *environmentUsecaseImpl) GetActiveSummary(ctx context.Context) (*entity.EnvironmentSummary, error) {
	return u.repo.GetActiveSummary(ctx)
}
