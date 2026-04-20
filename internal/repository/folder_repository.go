package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/folder"
	"PostmanJanai/ent/history"
	"PostmanJanai/ent/request"
	"PostmanJanai/ent/requestformfield"
	"PostmanJanai/ent/requestheader"
	"PostmanJanai/ent/requestqueryparam"
	"PostmanJanai/internal/entity"
	"context"
	"errors"
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

	// SearchByName returns folders whose name matches `query` (case-insensitive
	// substring). Capped at `limit` rows. The returned count is len(result); the
	// second return value signals truncation (true ⇒ there may be more matches).
	SearchByName(ctx context.Context, query string, limit int) ([]*entity.FolderItem, bool, error)

	// ListAllSkeleton returns every folder as (id, name, parent_id) tuples,
	// ordered so that callers can build the complete in-memory hierarchy. Used
	// by search to compute breadcrumb paths without recursive DB hits.
	ListAllSkeleton(ctx context.Context) ([]*entity.FolderItem, error)

	// MoveToParent re-parents a folder. `newParentID` empty string means root
	// (parent_id NULL). Caller must validate cycles and name uniqueness.
	MoveToParent(ctx context.Context, folderID string, newParentID *string) error

	// ReorderFolderSibling places folderID among siblings of parentID (empty = roots)
	// before insertBeforeID; empty insertBeforeID = append at end. Caller must ensure
	// folder is already under parentID (e.g. after MoveToParent).
	ReorderFolderSibling(ctx context.Context, folderID, parentID, insertBeforeID string) error
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
		SortOrder:   f.SortOrder,
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
	nextOrder, err := r.nextSortOrderAppend(ctx, item.ParentID)
	if err != nil {
		return "", err
	}
	b = b.SetSortOrder(nextOrder)
	f, err := b.Save(ctx)
	if err != nil {
		return "", err
	}
	return f.ID.String(), nil
}

func (r *folderRepo) nextSortOrderAppend(ctx context.Context, parentID *string) (int, error) {
	q := r.client.Folder.Query()
	if parentID == nil || strings.TrimSpace(*parentID) == "" {
		q = q.Where(folder.ParentIDIsNil())
	} else {
		pid, err := uuid.Parse(strings.TrimSpace(*parentID))
		if err != nil {
			return 0, err
		}
		q = q.Where(folder.ParentIDEQ(pid))
	}
	n, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}
	return n, nil
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
		Order(ent.Asc(folder.FieldSortOrder), ent.Asc(folder.FieldName)).
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
		Order(ent.Asc(folder.FieldSortOrder), ent.Asc(folder.FieldName)).
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

// DeleteByID removes a folder and its entire subtree (child folders + saved requests
// + per-request child rows) in a single transaction. History rows that reference any
// of the deleted folders / requests are preserved but their FK columns
// (root_folder_id, request_id) are nulled out — we want the snapshot to remain viewable
// even when the owning folder/request is gone.
func (r *folderRepo) DeleteByID(ctx context.Context, id string) error {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	folderIDs, err := r.collectSubtreeIDs(ctx, uid)
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

	requestIDs, err := tx.Request.Query().Where(request.FolderIDIn(folderIDs...)).IDs(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if len(requestIDs) > 0 {
		if _, err := tx.History.Update().
			Where(history.RequestIDIn(requestIDs...)).
			ClearRequestID().
			Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if _, err := tx.History.Update().
		Where(history.RootFolderIDIn(folderIDs...)).
		ClearRootFolderID().
		Save(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if len(requestIDs) > 0 {
		if _, err := tx.RequestHeader.Delete().Where(requestheader.RequestIDIn(requestIDs...)).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.RequestQueryParam.Delete().Where(requestqueryparam.RequestIDIn(requestIDs...)).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.RequestFormField.Delete().Where(requestformfield.RequestIDIn(requestIDs...)).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Request.Delete().Where(request.IDIn(requestIDs...)).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	// Bottom-up delete so children are removed before their parents.
	for i := len(folderIDs) - 1; i >= 0; i-- {
		if _, err := tx.Folder.Delete().Where(folder.IDEQ(folderIDs[i])).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// collectSubtreeIDs returns every folder UUID in the subtree rooted at `root`,
// ordered top-down (root first, deepest last). The caller deletes in reverse order.
func (r *folderRepo) collectSubtreeIDs(ctx context.Context, root uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{root}
	queue := []uuid.UUID{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		children, err := r.client.Folder.Query().
			Where(folder.ParentIDEQ(cur)).
			IDs(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, children...)
		queue = append(queue, children...)
	}
	return out, nil
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

func (r *folderRepo) SearchByName(ctx context.Context, query string, limit int) ([]*entity.FolderItem, bool, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, false, nil
	}
	if limit <= 0 {
		limit = 100
	}
	// +1 so we can detect truncation with a single query.
	rows, err := r.client.Folder.Query().
		Where(folder.NameContainsFold(q)).
		Order(ent.Asc(folder.FieldName)).
		Limit(limit + 1).
		All(ctx)
	if err != nil {
		return nil, false, err
	}
	truncated := len(rows) > limit
	if truncated {
		rows = rows[:limit]
	}
	out := make([]*entity.FolderItem, 0, len(rows))
	for _, f := range rows {
		out = append(out, entFolderToItem(f))
	}
	return out, truncated, nil
}

func (r *folderRepo) ListAllSkeleton(ctx context.Context) ([]*entity.FolderItem, error) {
	rows, err := r.client.Folder.Query().
		Select(folder.FieldID, folder.FieldName, folder.FieldParentID).
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

func (r *folderRepo) MoveToParent(ctx context.Context, folderID string, newParentID *string) error {
	uid, err := uuid.Parse(strings.TrimSpace(folderID))
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	var newParentUUID *uuid.UUID
	if newParentID == nil || strings.TrimSpace(*newParentID) == "" {
		if err := tx.Folder.UpdateOneID(uid).ClearParentID().Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	} else {
		pid, err := uuid.Parse(strings.TrimSpace(*newParentID))
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		newParentUUID = &pid
		if err := tx.Folder.UpdateOneID(uid).SetParentID(pid).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	next, err := r.nextSortOrderAppendTx(ctx, tx, newParentUUID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Folder.UpdateOneID(uid).SetSortOrder(next).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *folderRepo) nextSortOrderAppendTx(ctx context.Context, tx *ent.Tx, parentID *uuid.UUID) (int, error) {
	q := tx.Folder.Query()
	if parentID == nil {
		q = q.Where(folder.ParentIDIsNil())
	} else {
		q = q.Where(folder.ParentIDEQ(*parentID))
	}
	n, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}
	return n - 1, nil
}

func (r *folderRepo) ReorderFolderSibling(ctx context.Context, folderID, parentID, insertBeforeID string) error {
	fid, err := uuid.Parse(strings.TrimSpace(folderID))
	if err != nil {
		return err
	}
	var parentUUID *uuid.UUID
	if strings.TrimSpace(parentID) != "" {
		p, err := uuid.Parse(strings.TrimSpace(parentID))
		if err != nil {
			return err
		}
		parentUUID = &p
	}
	var insertBefore *uuid.UUID
	if strings.TrimSpace(insertBeforeID) != "" {
		b, err := uuid.Parse(strings.TrimSpace(insertBeforeID))
		if err != nil {
			return err
		}
		insertBefore = &b
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

	q := tx.Folder.Query()
	if parentUUID == nil {
		q = q.Where(folder.ParentIDIsNil())
	} else {
		q = q.Where(folder.ParentIDEQ(*parentUUID))
	}
	rows, err := q.Order(ent.Asc(folder.FieldSortOrder), ent.Asc(folder.FieldName)).All(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	foundMoving := false
	for _, row := range rows {
		ids = append(ids, row.ID)
		if row.ID == fid {
			foundMoving = true
		}
	}
	if !foundMoving {
		_ = tx.Rollback()
		return errFolderNotUnderParent
	}
	var without []uuid.UUID
	for _, id := range ids {
		if id != fid {
			without = append(without, id)
		}
	}
	insertIdx := len(without)
	if insertBefore != nil {
		found := -1
		for i, id := range without {
			if id == *insertBefore {
				found = i
				break
			}
		}
		if found < 0 {
			_ = tx.Rollback()
			return errInvalidReorderTarget
		}
		insertIdx = found
	}
	out := make([]uuid.UUID, 0, len(without)+1)
	out = append(out, without[:insertIdx]...)
	out = append(out, fid)
	out = append(out, without[insertIdx:]...)
	for i, id := range out {
		if err := tx.Folder.UpdateOneID(id).SetSortOrder(i).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

var (
	errInvalidReorderTarget = errors.New("insertBefore folder is not a sibling under this parent")
	errFolderNotUnderParent = errors.New("folder is not a child of the given parent")
)

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
