package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/folder"
	"PostmanJanai/ent/request"
	"PostmanJanai/internal/entity"
	"context"
	"strings"

	"github.com/google/uuid"
)

// FolderRepository persists nested folders (replaces workspace + collection).
type FolderRepository interface {
	Create(ctx context.Context, item *entity.FolderItem) (string, error)
	UpdateByID(ctx context.Context, item *entity.FolderItem) error
	GetByID(ctx context.Context, id string) (*entity.FolderItem, error)
	ListRoots(ctx context.Context) ([]*entity.FolderItem, error)
	ListChildren(ctx context.Context, parentID string) ([]*entity.FolderItem, error)
	DeleteByID(ctx context.Context, id string) error
	RootNameTaken(ctx context.Context, name string, excludeID *string) (bool, error)
	ChildNameTaken(ctx context.Context, parentID, name string, excludeID *string) (bool, error)
	ResolveRootID(ctx context.Context, folderID string) (string, error)
}

type folderRepo struct {
	client *ent.Client
}

func NewFolderRepository(client *ent.Client) FolderRepository {
	return &folderRepo{client: client}
}

func entFolderToItem(f *ent.Folder) *entity.FolderItem {
	out := &entity.FolderItem{
		ID:          f.ID.String(),
		Name:        f.Name,
		Description: f.Description,
		CreatedAt:   f.CreatedAt,
	}
	if f.ParentID != nil {
		s := f.ParentID.String()
		out.ParentID = &s
	}
	return out
}

func (r *folderRepo) Create(ctx context.Context, item *entity.FolderItem) (string, error) {
	b := r.client.Folder.Create().
		SetName(strings.TrimSpace(item.Name)).
		SetDescription(strings.TrimSpace(item.Description))
	if item.ParentID != nil && strings.TrimSpace(*item.ParentID) != "" {
		pid, err := uuid.Parse(strings.TrimSpace(*item.ParentID))
		if err != nil {
			return "", err
		}
		b = b.SetParentID(pid)
	}
	f, err := b.Save(ctx)
	if err != nil {
		return "", err
	}
	return f.ID.String(), nil
}

func (r *folderRepo) UpdateByID(ctx context.Context, item *entity.FolderItem) error {
	uid, err := uuid.Parse(strings.TrimSpace(item.ID))
	if err != nil {
		return err
	}
	return r.client.Folder.UpdateOneID(uid).
		SetName(strings.TrimSpace(item.Name)).
		SetDescription(strings.TrimSpace(item.Description)).
		Exec(ctx)
}

func (r *folderRepo) GetByID(ctx context.Context, id string) (*entity.FolderItem, error) {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	f, err := r.client.Folder.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	return entFolderToItem(f), nil
}

func (r *folderRepo) ListRoots(ctx context.Context) ([]*entity.FolderItem, error) {
	rows, err := r.client.Folder.Query().
		Where(folder.ParentIDIsNil()).
		Order(ent.Desc(folder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.FolderItem, 0, len(rows))
	for _, f := range rows {
		out = append(out, entFolderToItem(f))
	}
	return out, nil
}

func (r *folderRepo) ListChildren(ctx context.Context, parentID string) ([]*entity.FolderItem, error) {
	pid, err := uuid.Parse(strings.TrimSpace(parentID))
	if err != nil {
		return nil, err
	}
	rows, err := r.client.Folder.Query().
		Where(folder.ParentIDEQ(pid)).
		Order(ent.Asc(folder.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.FolderItem, 0, len(rows))
	for _, f := range rows {
		out = append(out, entFolderToItem(f))
	}
	return out, nil
}

func (r *folderRepo) DeleteByID(ctx context.Context, id string) error {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	subs, err := r.client.Folder.Query().Where(folder.ParentIDEQ(uid)).IDs(ctx)
	if err != nil {
		return err
	}
	for _, sid := range subs {
		if err := r.DeleteByID(ctx, sid.String()); err != nil {
			return err
		}
	}
	if _, err := r.client.Request.Delete().Where(request.FolderIDEQ(uid)).Exec(ctx); err != nil {
		return err
	}
	return r.client.Folder.DeleteOneID(uid).Exec(ctx)
}

func (r *folderRepo) RootNameTaken(ctx context.Context, name string, excludeID *string) (bool, error) {
	name = strings.TrimSpace(name)
	q := r.client.Folder.Query().Where(
		folder.ParentIDIsNil(),
		folder.NameEQ(name),
	)
	if excludeID != nil && strings.TrimSpace(*excludeID) != "" {
		ex, err := uuid.Parse(strings.TrimSpace(*excludeID))
		if err != nil {
			return false, err
		}
		q = q.Where(folder.IDNEQ(ex))
	}
	return q.Exist(ctx)
}

func (r *folderRepo) ChildNameTaken(ctx context.Context, parentID, name string, excludeID *string) (bool, error) {
	pid, err := uuid.Parse(strings.TrimSpace(parentID))
	if err != nil {
		return false, err
	}
	name = strings.TrimSpace(name)
	q := r.client.Folder.Query().Where(
		folder.ParentIDEQ(pid),
		folder.NameEQ(name),
	)
	if excludeID != nil && strings.TrimSpace(*excludeID) != "" {
		ex, err := uuid.Parse(strings.TrimSpace(*excludeID))
		if err != nil {
			return false, err
		}
		q = q.Where(folder.IDNEQ(ex))
	}
	return q.Exist(ctx)
}

func (r *folderRepo) ResolveRootID(ctx context.Context, folderID string) (string, error) {
	cur, err := uuid.Parse(strings.TrimSpace(folderID))
	if err != nil {
		return "", err
	}
	for {
		f, err := r.client.Folder.Get(ctx, cur)
		if err != nil {
			return "", err
		}
		if f.ParentID == nil {
			return f.ID.String(), nil
		}
		cur = *f.ParentID
	}
}
