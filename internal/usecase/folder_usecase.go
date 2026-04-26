package usecase

import (
	"PostmanJanai/ent"
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type FolderUsecase interface {
	CreateFolder(ctx context.Context, in *entity.CreateFolderInput) (*entity.FolderItem, *apperror.AppError)
	ListRootFolders(ctx context.Context) ([]*entity.FolderItem, error)
	ListChildFolders(ctx context.Context, parentID string) ([]*entity.FolderItem, error)
	UpdateFolder(ctx context.Context, id, name, desc string) error
	DeleteFolder(ctx context.Context, id string) error
	// MoveFolder re-parents a folder. `newParentID` empty string = move to root.
	MoveFolder(ctx context.Context, folderID, newParentID string) error
	// ReorderFolder moves folder among siblings of parentID (empty = roots).
	// insertBeforeID empty = append at end; otherwise insert before that sibling.
	ReorderFolder(ctx context.Context, folderID, parentID, insertBeforeID string) error
	DuplicateFolder(ctx context.Context, folderID string) (*entity.FolderItem, error)
}

type folderUsecaseImpl struct {
	folders  repository.FolderRepository
	requests repository.RequestRepository
}

func NewFolderUsecase(folders repository.FolderRepository) FolderUsecase {
	return &folderUsecaseImpl{folders: folders}
}

func NewFolderUsecaseWithRequests(folders repository.FolderRepository, requests repository.RequestRepository) FolderUsecase {
	return &folderUsecaseImpl{folders: folders, requests: requests}
}

func (u *folderUsecaseImpl) CreateFolder(ctx context.Context, in *entity.CreateFolderInput) (*entity.FolderItem, *apperror.AppError) {
	if in == nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInternal, errors.New("nil folder payload"))
	}
	name := strings.TrimSpace(in.Name)
	desc := strings.TrimSpace(in.Description)
	if name == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInternal, errors.New("empty folder name"))
	}
	if in.ParentID == nil || strings.TrimSpace(*in.ParentID) == "" {
		taken, err := u.folders.RootNameTaken(ctx, name, nil)
		if err != nil {
			return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
		}
		if taken {
			return nil, apperror.NewWithErrorDetail(constant.ErrFolderRootNameConflict, nil)
		}
	} else {
		pid := strings.TrimSpace(*in.ParentID)
		if _, err := uuid.Parse(pid); err != nil {
			return nil, apperror.NewWithErrorDetail(constant.ErrInternal, err)
		}
		if _, err := u.folders.GetByID(ctx, pid); err != nil {
			if ent.IsNotFound(err) {
				return nil, apperror.NewWithErrorDetail(constant.ErrFolderNotFound, nil)
			}
			return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
		}
		taken, err := u.folders.ChildNameTaken(ctx, pid, name, nil)
		if err != nil || taken {
			if taken {
				return nil, apperror.NewWithErrorDetail(constant.ErrFolderChildNameConflict, nil)
			}
			return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
		}
	}
	id, err := u.folders.Create(ctx, &entity.FolderItem{
		ParentID:    in.ParentID,
		Name:        name,
		Description: desc,
	})
	if err != nil {
		logger.L().ErrorContext(ctx, "CreateFolder save failed", "error", err)
		return nil, apperror.NewWithErrorDetail(constant.ErrFolderSaveFail, err)
	}
	item, err := u.folders.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrDatabase, err)
	}
	return item, nil
}

func (u *folderUsecaseImpl) ListRootFolders(ctx context.Context) ([]*entity.FolderItem, error) {
	return u.folders.ListRoots(ctx)
}

func (u *folderUsecaseImpl) ListChildFolders(ctx context.Context, parentID string) ([]*entity.FolderItem, error) {
	if _, err := uuid.Parse(strings.TrimSpace(parentID)); err != nil {
		return nil, err
	}
	if _, err := u.folders.GetByID(ctx, parentID); err != nil {
		return nil, err
	}
	return u.folders.ListChildren(ctx, parentID)
}

func (u *folderUsecaseImpl) UpdateFolder(ctx context.Context, id, name, desc string) error {
	name = strings.TrimSpace(name)
	desc = strings.TrimSpace(desc)
	if name == "" {
		return errors.New("empty folder name")
	}
	existing, err := u.folders.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ParentID == nil {
		taken, err := u.folders.RootNameTaken(ctx, name, &id)
		if err != nil {
			return err
		}
		if taken {
			return apperror.NewWithErrorDetail(constant.ErrFolderRootNameConflict, nil)
		}
	} else {
		taken, err := u.folders.ChildNameTaken(ctx, *existing.ParentID, name, &id)
		if err != nil {
			return err
		}
		if taken {
			return apperror.NewWithErrorDetail(constant.ErrFolderChildNameConflict, nil)
		}
	}
	return u.folders.UpdateByID(ctx, &entity.FolderItem{
		ID:          id,
		ParentID:    existing.ParentID,
		Name:        name,
		Description: desc,
	})
}

func (u *folderUsecaseImpl) DeleteFolder(ctx context.Context, id string) error {
	if _, err := u.folders.GetByID(ctx, id); err != nil {
		return err
	}
	return u.folders.DeleteByID(ctx, id)
}

// isDescendantOf returns true if `descendantID` is `ancestorID` itself or any
// nested folder under it (walk upward from descendant toward root).
func (u *folderUsecaseImpl) isDescendantOf(ctx context.Context, ancestorID, descendantID string) bool {
	cur := strings.TrimSpace(descendantID)
	target := strings.TrimSpace(ancestorID)
	if cur == "" || target == "" {
		return false
	}
	for i := 0; i < 100000; i++ {
		if cur == target {
			return true
		}
		f, err := u.folders.GetByID(ctx, cur)
		if err != nil || f.ParentID == nil {
			return false
		}
		cur = *f.ParentID
	}
	return false
}

func (u *folderUsecaseImpl) MoveFolder(ctx context.Context, folderID, newParentID string) error {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return errors.New("empty folder id")
	}
	moving, err := u.folders.GetByID(ctx, folderID)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(moving.Name)
	np := strings.TrimSpace(newParentID)
	if np == "" {
		taken, err := u.folders.RootNameTaken(ctx, name, &folderID)
		if err != nil {
			return err
		}
		if taken {
			return apperror.NewWithErrorDetail(constant.ErrFolderRootNameConflict, nil)
		}
		return u.folders.MoveToParent(ctx, folderID, nil)
	}
	if np == folderID {
		return errors.New("cannot move folder into itself")
	}
	if u.isDescendantOf(ctx, folderID, np) {
		return errors.New("cannot move folder into its own subtree")
	}
	if _, err := u.folders.GetByID(ctx, np); err != nil {
		return err
	}
	taken, err := u.folders.ChildNameTaken(ctx, np, name, &folderID)
	if err != nil {
		return err
	}
	if taken {
		return apperror.NewWithErrorDetail(constant.ErrFolderChildNameConflict, nil)
	}
	return u.folders.MoveToParent(ctx, folderID, &np)
}

func parentsEqualFolder(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return strings.TrimSpace(*a) == strings.TrimSpace(*b)
}

func (u *folderUsecaseImpl) ReorderFolder(ctx context.Context, folderID, parentID, insertBeforeID string) error {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return errors.New("empty folder id")
	}
	moving, err := u.folders.GetByID(ctx, folderID)
	if err != nil {
		return err
	}
	pid := strings.TrimSpace(parentID)
	var targetParent *string
	if pid == "" {
		targetParent = nil
	} else {
		if _, err := u.folders.GetByID(ctx, pid); err != nil {
			return err
		}
		targetParent = &pid
	}
	ib := strings.TrimSpace(insertBeforeID)
	if ib != "" {
		before, err := u.folders.GetByID(ctx, ib)
		if err != nil {
			return err
		}
		if !parentsEqualFolder(before.ParentID, targetParent) {
			return errors.New("insertBefore must be a sibling under the target parent")
		}
	}
	if !parentsEqualFolder(moving.ParentID, targetParent) {
		if err := u.MoveFolder(ctx, folderID, pid); err != nil {
			return err
		}
	}
	return u.folders.ReorderFolderSibling(ctx, folderID, pid, ib)
}

func (u *folderUsecaseImpl) DuplicateFolder(ctx context.Context, folderID string) (*entity.FolderItem, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return nil, errors.New("empty folder id")
	}
	if u.requests == nil {
		return nil, errors.New("request repository is required to duplicate folders")
	}
	original, err := u.folders.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	name, err := u.uniqueFolderCopyName(ctx, original.ParentID, original.Name)
	if err != nil {
		return nil, err
	}
	rootCopy, appErr := u.CreateFolder(ctx, &entity.CreateFolderInput{
		ParentID:    original.ParentID,
		Name:        name,
		Description: original.Description,
	})
	if appErr != nil {
		return nil, appErr
	}
	if err := u.copyFolderContents(ctx, original.ID, rootCopy.ID); err != nil {
		_ = u.folders.DeleteByID(ctx, rootCopy.ID)
		return nil, err
	}
	return rootCopy, nil
}

func (u *folderUsecaseImpl) copyFolderContents(ctx context.Context, sourceFolderID, destFolderID string) error {
	reqs, err := u.requests.ListByFolder(ctx, sourceFolderID)
	if err != nil {
		return err
	}
	for _, summary := range reqs {
		full, err := u.requests.GetByID(ctx, summary.ID)
		if err != nil {
			return err
		}
		copyReq := cloneSavedRequestForCreate(full)
		copyReq.FolderID = destFolderID
		copyReq.Name = full.Name
		if _, err := u.requests.CreateFull(ctx, copyReq); err != nil {
			return err
		}
	}

	children, err := u.folders.ListChildren(ctx, sourceFolderID)
	if err != nil {
		return err
	}
	for _, child := range children {
		parentID := destFolderID
		childCopy, appErr := u.CreateFolder(ctx, &entity.CreateFolderInput{
			ParentID:    &parentID,
			Name:        child.Name,
			Description: child.Description,
		})
		if appErr != nil {
			return appErr
		}
		if err := u.copyFolderContents(ctx, child.ID, childCopy.ID); err != nil {
			return err
		}
	}
	return nil
}

func (u *folderUsecaseImpl) uniqueFolderCopyName(ctx context.Context, parentID *string, base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Folder"
	}
	for i := 1; i < 10000; i++ {
		name := base + " (copy)"
		if i > 1 {
			name = base + " (copy " + strconv.Itoa(i) + ")"
		}
		var (
			taken bool
			err   error
		)
		if parentID == nil || strings.TrimSpace(*parentID) == "" {
			taken, err = u.folders.RootNameTaken(ctx, name, nil)
		} else {
			taken, err = u.folders.ChildNameTaken(ctx, *parentID, name, nil)
		}
		if err != nil {
			return "", err
		}
		if !taken {
			return name, nil
		}
	}
	if parentID == nil {
		return "", apperror.NewWithErrorDetail(constant.ErrFolderRootNameConflict, nil)
	}
	return "", apperror.NewWithErrorDetail(constant.ErrFolderChildNameConflict, nil)
}
