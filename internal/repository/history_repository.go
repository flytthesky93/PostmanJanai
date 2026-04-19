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
	ListSummaries(ctx context.Context, rootFolderID *string) ([]entity.HistorySummary, error)
	GetByID(ctx context.Context, id string) (*entity.HistoryItem, error)
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

func (r *historyRepo) ListSummaries(ctx context.Context, rootFolderID *string) ([]entity.HistorySummary, error) {
	q := r.client.History.Query()
	if rootFolderID != nil {
		s := strings.TrimSpace(*rootFolderID)
		if s != "" {
			uid, err := uuid.Parse(s)
			if err != nil {
				return nil, err
			}
			q = q.Where(history.RootFolderIDEQ(uid))
		}
	}
	rows, err := q.
		Order(ent.Desc(history.FieldCreatedAt)).
		Select(
			history.FieldID,
			history.FieldRootFolderID,
			history.FieldRequestID,
			history.FieldMethod,
			history.FieldURL,
			history.FieldStatusCode,
			history.FieldDurationMs,
			history.FieldResponseSizeBytes,
			history.FieldCreatedAt,
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.HistorySummary, 0, len(rows))
	for _, row := range rows {
		var rf *string
		if row.RootFolderID != nil {
			s := row.RootFolderID.String()
			rf = &s
		}
		var rq *string
		if row.RequestID != nil {
			s := row.RequestID.String()
			rq = &s
		}
		out = append(out, entity.HistorySummary{
			ID:                row.ID.String(),
			RootFolderID:      rf,
			RequestID:         rq,
			Method:            row.Method,
			URL:               row.URL,
			StatusCode:        row.StatusCode,
			DurationMs:        row.DurationMs,
			ResponseSizeBytes: row.ResponseSizeBytes,
			CreatedAt:         row.CreatedAt,
		})
	}
	return out, nil
}

func (r *historyRepo) GetByID(ctx context.Context, id string) (*entity.HistoryItem, error) {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	row, err := r.client.History.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	var rf *string
	if row.RootFolderID != nil {
		s := row.RootFolderID.String()
		rf = &s
	}
	var rq *string
	if row.RequestID != nil {
		s := row.RequestID.String()
		rq = &s
	}
	return &entity.HistoryItem{
		ID:                  row.ID.String(),
		RootFolderID:        rf,
		RequestID:           rq,
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
	}, nil
}

func (r *historyRepo) DeleteByID(ctx context.Context, id string) error {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	return r.client.History.DeleteOneID(uid).Exec(ctx)
}
