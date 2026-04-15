package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/workspace"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"context"

	"github.com/google/uuid"
)

type WorkspaceRepository interface {
	Save(ctx context.Context, workspace *entity.WorkspaceItem) (string, error)
	UpdateByID(ctx context.Context, workspace *entity.WorkspaceItem) error
	GetAll(ctx context.Context) ([]*entity.WorkspaceItem, error)
	DeleteByID(ctx context.Context, id string) error
	GetByName(ctx context.Context, name string) (*ent.Workspace, error)
}

type workspaceRepo struct {
	client *ent.Client
}

func (r *workspaceRepo) GetByName(ctx context.Context, name string) (*ent.Workspace, error) {
	logger.D().InfoContext(ctx, "Repository.GetByName called", "name", name)
	ws, err := r.client.Workspace.Query().Where(workspace.WorkspaceNameEQ(name)).Only(ctx)
	if err != nil {
		logger.D().ErrorContext(ctx, "Repository.GetByName failed", "name", name, "error", err)
	}
	return ws, err
}

func NewWorkspaceRepository(client *ent.Client) WorkspaceRepository {
	return &workspaceRepo{client: client}
}

func (r *workspaceRepo) Save(ctx context.Context, item *entity.WorkspaceItem) (string, error) {
	logger.D().InfoContext(ctx, "Repository.Save called", "workspace_name", item.WorkspaceName)
	builder := r.client.Workspace.
		Create().
		SetWorkspaceName(item.WorkspaceName).
		SetWorkspaceDescription(item.WorkspaceDescription)
	if !item.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(item.CreatedAt)
	}
	ws, err := builder.Save(ctx)
	if err != nil {
		logger.D().ErrorContext(ctx, "Repository.Save failed", "error", err)
		return "", err
	}
	logger.D().InfoContext(ctx, "Repository.Save success", "id", ws.ID, "workspace_name", ws.WorkspaceName)
	return ws.ID.String(), err
}

func (r *workspaceRepo) UpdateByID(ctx context.Context, item *entity.WorkspaceItem) error {
	id, err := uuid.Parse(item.ID)
	if err != nil {
		return err
	}
	update := r.client.Workspace.
		UpdateOneID(id).
		SetWorkspaceName(item.WorkspaceName).
		SetWorkspaceDescription(item.WorkspaceDescription)
	if err := update.Exec(ctx); err != nil {
		logger.D().ErrorContext(ctx, "Repository.UpdateByID failed", "id", item.ID, "error", err)
		return err
	}
	logger.D().InfoContext(ctx, "Repository.UpdateByID success", "id", item.ID)
	return nil
}

func (r *workspaceRepo) GetAll(ctx context.Context) ([]*entity.WorkspaceItem, error) {
	rows, err := r.client.Workspace.
		Query().
		Order(ent.Desc(workspace.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		logger.D().ErrorContext(ctx, "Repository.GetAll failed", "error", err)
		return nil, err
	}

	var result []*entity.WorkspaceItem
	for _, row := range rows {
		result = append(result, &entity.WorkspaceItem{
			ID:                   row.ID.String(),
			WorkspaceName:        row.WorkspaceName,
			WorkspaceDescription: row.WorkspaceDescription,
			CreatedAt:            row.CreatedAt,
		})
	}

	return result, nil
}

func (r *workspaceRepo) DeleteByID(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	if err := r.client.Workspace.DeleteOneID(uid).Exec(ctx); err != nil {
		logger.D().ErrorContext(ctx, "Repository.DeleteByID failed", "id", id, "error", err)
		return err
	}
	logger.D().InfoContext(ctx, "Repository.DeleteByID success", "id", id)
	return nil
}
