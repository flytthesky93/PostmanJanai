package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/workspace"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"context"
)

type WorkspaceRepository interface {
	Save(ctx context.Context, workspace *entity.WorkspaceItem) (int, error)
	GetAll(ctx context.Context) ([]*entity.WorkspaceItem, error)
	DeleteByID(ctx context.Context, id int) error
	GetByName(ctx context.Context, name string) (*ent.Workspace, error)
}

type workspaceRepo struct {
	client *ent.Client
}

func (r *workspaceRepo) GetByName(ctx context.Context, name string) (*ent.Workspace, error) {
	return r.client.Workspace.Query().Where(workspace.WorkspaceNameEQ(name)).Only(ctx)
}

func NewWorkspaceRepository(client *ent.Client) WorkspaceRepository {
	return &workspaceRepo{client: client}
}

// Save lưu một bản ghi lịch sử mới
func (r *workspaceRepo) Save(ctx context.Context, item *entity.WorkspaceItem) (int, error) {
	ws, err := r.client.Workspace.
		Create().
		SetWorkspaceName(item.WorkspaceName).
		SetWorkspaceDescription(item.WorkspaceDescription).
		SetCreatedAt(item.CreatedAt).
		Save(ctx)
	if err != nil {
		logger.L().ErrorContext(ctx, "Save workspace error, err: %v", err)
		return -1, err
	}
	return ws.ID, err
}

// GetAll lấy toàn bộ lịch sử và chuyển đổi sang Entity sạch
func (r *workspaceRepo) GetAll(ctx context.Context) ([]*entity.WorkspaceItem, error) {
	// Query từ DB (sắp xếp mới nhất lên đầu)
	rows, err := r.client.Workspace.
		Query().
		Order(ent.Desc(workspace.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, err
	}

	// Mapping từ Ent sang Entity
	var result []*entity.WorkspaceItem
	for _, row := range rows {
		result = append(result, &entity.WorkspaceItem{
			ID:                   row.ID,
			WorkspaceName:        row.WorkspaceName,
			WorkspaceDescription: row.WorkspaceDescription,
			CreatedAt:            row.CreatedAt,
		})
	}

	return result, nil
}

// DeleteByID xóa một bản ghi
func (r *workspaceRepo) DeleteByID(ctx context.Context, id int) error {
	return r.client.Workspace.DeleteOneID(id).Exec(ctx)
}
