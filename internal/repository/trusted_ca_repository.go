package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/trustedca"
	"context"

	"github.com/google/uuid"
)

// TrustedCARepository persists user-imported CA certificates (PEM).
type TrustedCARepository interface {
	ListAll(ctx context.Context) ([]*ent.TrustedCA, error)
	ListEnabledPEMs(ctx context.Context) ([][]byte, error)
	Create(ctx context.Context, label, pemContent string) (*ent.TrustedCA, error)
	SetEnabled(ctx context.Context, id string, enabled bool) error
	Delete(ctx context.Context, id string) error
}

type trustedCARepo struct {
	client *ent.Client
}

func NewTrustedCARepository(client *ent.Client) TrustedCARepository {
	return &trustedCARepo{client: client}
}

func (r *trustedCARepo) ListAll(ctx context.Context) ([]*ent.TrustedCA, error) {
	return r.client.TrustedCA.Query().
		Order(ent.Desc(trustedca.FieldCreatedAt)).
		All(ctx)
}

func (r *trustedCARepo) ListEnabledPEMs(ctx context.Context) ([][]byte, error) {
	list, err := r.client.TrustedCA.Query().
		Where(trustedca.Enabled(true)).
		Order(ent.Asc(trustedca.FieldLabel)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var out [][]byte
	for _, row := range list {
		out = append(out, []byte(row.PemContent))
	}
	return out, nil
}

func (r *trustedCARepo) Create(ctx context.Context, label, pemContent string) (*ent.TrustedCA, error) {
	return r.client.TrustedCA.Create().
		SetLabel(label).
		SetPemContent(pemContent).
		SetEnabled(true).
		Save(ctx)
}

func (r *trustedCARepo) SetEnabled(ctx context.Context, id string, enabled bool) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.client.TrustedCA.UpdateOneID(uid).SetEnabled(enabled).Exec(ctx)
}

func (r *trustedCARepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.client.TrustedCA.DeleteOneID(uid).Exec(ctx)
}
