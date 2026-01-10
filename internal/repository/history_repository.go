package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/history"
	"PostmanJanai/internal/entity"
	"context"
)

type HistoryRepository interface {
	Save(ctx context.Context, history *entity.HistoryItem) error
	GetAll(ctx context.Context) ([]entity.HistoryItem, error)
	DeleteByID(ctx context.Context, id int) error
}

type historyRepo struct {
	client *ent.Client
}

func NewHistoryRepository(client *ent.Client) HistoryRepository {
	return &historyRepo{client: client}
}

// Save lưu một bản ghi lịch sử mới
func (r *historyRepo) Save(ctx context.Context, item *entity.HistoryItem) error {
	_, err := r.client.History.
		Create().
		SetMethod(item.Method).
		SetURL(item.URL).
		SetStatusCode(item.StatusCode).
		SetRequestBody(item.RequestBody).
		SetResponseBody(item.ResponseBody).
		SetCreatedAt(item.CreatedAt).
		Save(ctx)
	return err
}

// GetAll lấy toàn bộ lịch sử và chuyển đổi sang Entity sạch
func (r *historyRepo) GetAll(ctx context.Context) ([]entity.HistoryItem, error) {
	// Query từ DB (sắp xếp mới nhất lên đầu)
	rows, err := r.client.History.
		Query().
		Order(ent.Desc(history.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, err
	}

	// Mapping từ Ent sang Entity
	var result []entity.HistoryItem
	for _, row := range rows {
		result = append(result, entity.HistoryItem{
			ID:           row.ID,
			Method:       row.Method,
			URL:          row.URL,
			StatusCode:   row.StatusCode,
			RequestBody:  row.RequestBody,
			ResponseBody: row.ResponseBody,
			CreatedAt:    row.CreatedAt,
		})
	}

	return result, nil
}

// DeleteByID xóa một bản ghi
func (r *historyRepo) DeleteByID(ctx context.Context, id int) error {
	return r.client.History.DeleteOneID(id).Exec(ctx)
}
