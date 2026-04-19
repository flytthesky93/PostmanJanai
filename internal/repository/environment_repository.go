package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/environment"
	"PostmanJanai/ent/environmentvariable"
	"PostmanJanai/internal/entity"
	"context"
	"strings"

	"github.com/google/uuid"
)

// EnvironmentRepository persists environments and their variables.
type EnvironmentRepository interface {
	ListSummaries(ctx context.Context) ([]entity.EnvironmentSummary, error)
	GetFull(ctx context.Context, id string) (*entity.EnvironmentFull, error)
	Create(ctx context.Context, name, description string) (*entity.EnvironmentFull, error)
	UpdateMeta(ctx context.Context, id, name, description string) error
	Delete(ctx context.Context, id string) error
	NameTaken(ctx context.Context, name string, excludeID *string) (bool, error)
	SaveVariables(ctx context.Context, envID string, rows []entity.EnvVariableInput) error
	SetActive(ctx context.Context, id string) error
	ClearActive(ctx context.Context) error
	GetActiveSummary(ctx context.Context) (*entity.EnvironmentSummary, error)
}

type environmentRepo struct {
	client *ent.Client
}

func NewEnvironmentRepository(client *ent.Client) EnvironmentRepository {
	return &environmentRepo{client: client}
}

func entEnvToSummary(e *ent.Environment) entity.EnvironmentSummary {
	return entity.EnvironmentSummary{
		ID:          e.ID.String(),
		Name:        e.Name,
		Description: e.Description,
		IsActive:    e.IsActive,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func (r *environmentRepo) ListSummaries(ctx context.Context) ([]entity.EnvironmentSummary, error) {
	list, err := r.client.Environment.Query().
		Order(ent.Desc(environment.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.EnvironmentSummary, 0, len(list))
	for _, e := range list {
		out = append(out, entEnvToSummary(e))
	}
	return out, nil
}

func (r *environmentRepo) GetFull(ctx context.Context, id string) (*entity.EnvironmentFull, error) {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	e, err := r.client.Environment.Query().
		Where(environment.IDEQ(uid)).
		WithEnvironmentVariables(func(q *ent.EnvironmentVariableQuery) {
			q.Order(ent.Asc(environmentvariable.FieldSortOrder), ent.Asc(environmentvariable.FieldID))
		}).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	sum := entEnvToSummary(e)
	vars := make([]entity.EnvironmentVariableRow, 0, len(e.Edges.EnvironmentVariables))
	for _, v := range e.Edges.EnvironmentVariables {
		vars = append(vars, entity.EnvironmentVariableRow{
			ID:        v.ID.String(),
			Key:       v.Key,
			Value:     v.Value,
			Enabled:   v.Enabled,
			SortOrder: v.SortOrder,
		})
	}
	return &entity.EnvironmentFull{EnvironmentSummary: sum, Variables: vars}, nil
}

func (r *environmentRepo) Create(ctx context.Context, name, description string) (*entity.EnvironmentFull, error) {
	n := strings.TrimSpace(name)
	d := strings.TrimSpace(description)
	e, err := r.client.Environment.Create().
		SetName(n).
		SetDescription(d).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.GetFull(ctx, e.ID.String())
}

func (r *environmentRepo) UpdateMeta(ctx context.Context, id, name, description string) error {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	return r.client.Environment.UpdateOneID(uid).
		SetName(strings.TrimSpace(name)).
		SetDescription(strings.TrimSpace(description)).
		Exec(ctx)
}

func (r *environmentRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	_, err = r.client.EnvironmentVariable.Delete().
		Where(environmentvariable.EnvironmentIDEQ(uid)).
		Exec(ctx)
	if err != nil {
		return err
	}
	return r.client.Environment.DeleteOneID(uid).Exec(ctx)
}

func (r *environmentRepo) NameTaken(ctx context.Context, name string, excludeID *string) (bool, error) {
	n := strings.TrimSpace(name)
	q := r.client.Environment.Query().Where(environment.NameEQ(n))
	if excludeID != nil {
		if ex, err := uuid.Parse(strings.TrimSpace(*excludeID)); err == nil {
			q = q.Where(environment.IDNEQ(ex))
		}
	}
	cnt, err := q.Count(ctx)
	return cnt > 0, err
}

func (r *environmentRepo) SaveVariables(ctx context.Context, envID string, rows []entity.EnvVariableInput) error {
	uid, err := uuid.Parse(strings.TrimSpace(envID))
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	_, err = tx.EnvironmentVariable.Delete().
		Where(environmentvariable.EnvironmentIDEQ(uid)).
		Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	for i, row := range rows {
		k := strings.TrimSpace(row.Key)
		if k == "" {
			continue
		}
		so := row.SortOrder
		if so == 0 {
			so = i
		}
		_, err = tx.EnvironmentVariable.Create().
			SetEnvironmentID(uid).
			SetKey(k).
			SetValue(row.Value).
			SetEnabled(row.Enabled).
			SetSortOrder(so).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *environmentRepo) SetActive(ctx context.Context, id string) error {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	_, err = tx.Environment.Update().SetIsActive(false).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = tx.Environment.UpdateOneID(uid).SetIsActive(true).Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *environmentRepo) ClearActive(ctx context.Context) error {
	_, err := r.client.Environment.Update().SetIsActive(false).Save(ctx)
	return err
}

func (r *environmentRepo) GetActiveSummary(ctx context.Context) (*entity.EnvironmentSummary, error) {
	e, err := r.client.Environment.Query().
		Where(environment.IsActive(true)).
		Order(ent.Desc(environment.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	s := entEnvToSummary(e)
	return &s, nil
}
