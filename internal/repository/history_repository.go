package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/history"
	"PostmanJanai/internal/entity"
	"context"
	"strings"

	"github.com/google/uuid"
)

type HistoryRepository interface {
	Save(ctx context.Context, item *entity.HistoryItem) error
	GetAll(ctx context.Context) ([]entity.HistoryItem, error)
	DeleteByID(ctx context.Context, id string) error
}

type historyRepo struct {
	client *ent.Client
}

func NewHistoryRepository(client *ent.Client) HistoryRepository {
	return &historyRepo{client: client}
}

func (r *historyRepo) Save(ctx context.Context, item *entity.HistoryItem) error {
	b := r.client.History.Create().
		SetMethod(item.Method).
		SetURL(item.URL).
		SetStatusCode(item.StatusCode).
		SetNillableDurationMs(item.DurationMs).
		SetNillableResponseSizeBytes(item.ResponseSizeBytes).
		SetNillableRequestHeadersJSON(item.RequestHeadersJSON).
		SetNillableResponseHeadersJSON(item.ResponseHeadersJSON).
		SetNillableRequestBody(item.RequestBody).
		SetNillableResponseBody(item.ResponseBody)
	if item.RootFolderID != nil {
		if s := strings.TrimSpace(*item.RootFolderID); s != "" {
			if uid, err := uuid.Parse(s); err == nil {
				b = b.SetRootFolderID(uid)
			}
		}
	}
	if item.RequestID != nil {
		if s := strings.TrimSpace(*item.RequestID); s != "" {
			if uid, err := uuid.Parse(s); err == nil {
				b = b.SetRequestID(uid)
			}
		}
	}
	if !item.CreatedAt.IsZero() {
		b = b.SetCreatedAt(item.CreatedAt)
	}
	_, err := b.Save(ctx)
	return err
}

func (r *historyRepo) GetAll(ctx context.Context) ([]entity.HistoryItem, error) {
	rows, err := r.client.History.
		Query().
		Order(ent.Desc(history.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, err
	}

	var result []entity.HistoryItem
	for _, row := range rows {
		var wsID *string
		if row.RootFolderID != nil {
			s := row.RootFolderID.String()
			wsID = &s
		}
		var reqID *string
		if row.RequestID != nil {
			s := row.RequestID.String()
			reqID = &s
		}
		result = append(result, entity.HistoryItem{
			ID:                  row.ID.String(),
			RootFolderID:      wsID,
			RequestID:           reqID,
			Method:              row.Method,
			URL:                 row.URL,
			StatusCode:          row.StatusCode,
			DurationMs:          row.DurationMs,
			ResponseSizeBytes:   row.ResponseSizeBytes,
			RequestHeadersJSON:  row.RequestHeadersJSON,
			ResponseHeadersJSON: row.ResponseHeadersJSON,
			RequestBody:         row.RequestBody,
			ResponseBody:        row.ResponseBody,
			CreatedAt:           row.CreatedAt,
		})
	}

	return result, nil
}

func (r *historyRepo) DeleteByID(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.client.History.DeleteOneID(uid).Exec(ctx)
}
