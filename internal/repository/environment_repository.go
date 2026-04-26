package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/environment"
	"PostmanJanai/ent/environmentvariable"
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/service"
	"context"
	"strings"
	"time"

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
	// ActiveVariableMap returns enabled key → value for the currently active environment (empty map if none).
	ActiveVariableMap(ctx context.Context) (map[string]string, error)
	// ActiveSecretPlaintexts returns decrypted plaintext values of **enabled secret** variables
	// in the active environment — used to redact them from history / snippets.
	ActiveSecretPlaintexts(ctx context.Context) ([]string, error)

	// UpsertActiveVariable writes a single key/value into the currently active environment.
	// Phase 8 — used by capture rules.
	//
	// Behaviour:
	//   - returns (false, nil) if there is no active environment (silent no-op).
	//   - if the key already exists, only `value` is overwritten (kind / enabled / order preserved).
	//   - new keys are inserted as plain (non-secret) and enabled, appended after existing rows.
	UpsertActiveVariable(ctx context.Context, key, value string) (bool, error)
}

type environmentRepo struct {
	client *ent.Client
	cipher *service.SecretCipher
}

func NewEnvironmentRepository(client *ent.Client, cipher *service.SecretCipher) EnvironmentRepository {
	return &environmentRepo{client: client, cipher: cipher}
}

func normalizeEnvVarKind(k string) string {
	s := strings.ToLower(strings.TrimSpace(k))
	if s == constant.EnvVarKindSecret {
		return constant.EnvVarKindSecret
	}
	return constant.EnvVarKindPlain
}

func (r *environmentRepo) decryptStoredValue(stored string) (string, error) {
	if r.cipher == nil {
		return stored, nil
	}
	return r.cipher.Decrypt(stored)
}

func (r *environmentRepo) encryptForStore(kind, plain string) (string, error) {
	if kind != constant.EnvVarKindSecret {
		return plain, nil
	}
	if r.cipher == nil {
		return plain, nil
	}
	return r.cipher.Encrypt(plain)
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
		kind := normalizeEnvVarKind(v.Kind)
		val := v.Value
		if dec, err := r.decryptStoredValue(val); err == nil {
			val = dec
		}
		vars = append(vars, entity.EnvironmentVariableRow{
			ID:        v.ID.String(),
			Key:       v.Key,
			Value:     val,
			Kind:      kind,
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
		kind := normalizeEnvVarKind(row.Kind)
		stored, encErr := r.encryptForStore(kind, row.Value)
		if encErr != nil {
			_ = tx.Rollback()
			return encErr
		}
		_, err = tx.EnvironmentVariable.Create().
			SetEnvironmentID(uid).
			SetKey(k).
			SetValue(stored).
			SetKind(kind).
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

func (r *environmentRepo) ActiveVariableMap(ctx context.Context) (map[string]string, error) {
	sum, err := r.GetActiveSummary(ctx)
	if err != nil {
		return nil, err
	}
	if sum == nil {
		return map[string]string{}, nil
	}
	uid, err := uuid.Parse(strings.TrimSpace(sum.ID))
	if err != nil {
		return map[string]string{}, nil
	}
	list, err := r.client.EnvironmentVariable.Query().
		Where(
			environmentvariable.EnvironmentIDEQ(uid),
			environmentvariable.Enabled(true),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(list))
	for _, v := range list {
		k := strings.TrimSpace(v.Key)
		if k == "" {
			continue
		}
		val := v.Value
		if dec, err := r.decryptStoredValue(val); err == nil {
			val = dec
		}
		m[k] = val
	}
	return m, nil
}

func (r *environmentRepo) UpsertActiveVariable(ctx context.Context, key, value string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, nil
	}
	sum, err := r.GetActiveSummary(ctx)
	if err != nil {
		return false, err
	}
	if sum == nil {
		return false, nil
	}
	envID, err := uuid.Parse(strings.TrimSpace(sum.ID))
	if err != nil {
		return false, err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	existing, err := tx.EnvironmentVariable.Query().
		Where(
			environmentvariable.EnvironmentIDEQ(envID),
			environmentvariable.KeyEQ(key),
		).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		_ = tx.Rollback()
		return false, err
	}
	if existing != nil {
		stored, encErr := r.encryptForStore(existing.Kind, value)
		if encErr != nil {
			_ = tx.Rollback()
			return false, encErr
		}
		if err := tx.EnvironmentVariable.UpdateOneID(existing.ID).SetValue(stored).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return false, err
		}
	} else {
		next, err := tx.EnvironmentVariable.Query().
			Where(environmentvariable.EnvironmentIDEQ(envID)).
			Count(ctx)
		if err != nil {
			_ = tx.Rollback()
			return false, err
		}
		stored, encErr := r.encryptForStore(constant.EnvVarKindPlain, value)
		if encErr != nil {
			_ = tx.Rollback()
			return false, encErr
		}
		_, err = tx.EnvironmentVariable.Create().
			SetEnvironmentID(envID).
			SetKey(key).
			SetValue(stored).
			SetKind(constant.EnvVarKindPlain).
			SetEnabled(true).
			SetSortOrder(next).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return false, err
		}
	}
	if err := tx.Environment.UpdateOneID(envID).SetUpdatedAt(time.Now()).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *environmentRepo) ActiveSecretPlaintexts(ctx context.Context) ([]string, error) {
	sum, err := r.GetActiveSummary(ctx)
	if err != nil {
		return nil, err
	}
	if sum == nil {
		return nil, nil
	}
	uid, err := uuid.Parse(strings.TrimSpace(sum.ID))
	if err != nil {
		return nil, nil
	}
	list, err := r.client.EnvironmentVariable.Query().
		Where(
			environmentvariable.EnvironmentIDEQ(uid),
			environmentvariable.Enabled(true),
			environmentvariable.KindEQ(constant.EnvVarKindSecret),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, v := range list {
		plain, err := r.decryptStoredValue(v.Value)
		if err != nil {
			continue
		}
		t := strings.TrimSpace(plain)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
