package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/setting"
	"context"
	"strings"
)

// SettingsRepository persists opaque key/value rows (proxy mode, proxy URL, …).
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

type settingsRepo struct {
	client *ent.Client
}

func NewSettingsRepository(client *ent.Client) SettingsRepository {
	return &settingsRepo{client: client}
}

func (r *settingsRepo) Get(ctx context.Context, key string) (string, error) {
	k := strings.TrimSpace(key)
	if k == "" {
		return "", nil
	}
	s, err := r.client.Setting.Query().
		Where(setting.KeyEQ(k)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return s.Value, nil
}

func (r *settingsRepo) Set(ctx context.Context, key, value string) error {
	k := strings.TrimSpace(key)
	if k == "" {
		return nil
	}
	existing, err := r.client.Setting.Query().
		Where(setting.KeyEQ(k)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if ent.IsNotFound(err) {
		_, err = r.client.Setting.Create().
			SetKey(k).
			SetValue(value).
			Save(ctx)
		return err
	}
	return r.client.Setting.UpdateOneID(existing.ID).
		SetValue(value).
		Exec(ctx)
}

func (r *settingsRepo) Delete(ctx context.Context, key string) error {
	k := strings.TrimSpace(key)
	if k == "" {
		return nil
	}
	_, err := r.client.Setting.Delete().
		Where(setting.KeyEQ(k)).
		Exec(ctx)
	return err
}
