package usecase

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"context"
	"fmt"
	"strings"
)

// ImportUsecase persists an already-parsed ImportedCollection (from internal/service importers)
// into folders + saved requests (+ optional environment) in one batch.
type ImportUsecase interface {
	PersistCollection(ctx context.Context, col *entity.ImportedCollection, opts entity.ImportOptions) (*entity.ImportResult, error)
}

type importUsecaseImpl struct {
	folders repository.FolderRepository
	reqs    repository.RequestRepository
	envs    repository.EnvironmentRepository
}

func NewImportUsecase(
	folders repository.FolderRepository,
	reqs repository.RequestRepository,
	envs repository.EnvironmentRepository,
) ImportUsecase {
	return &importUsecaseImpl{folders: folders, reqs: reqs, envs: envs}
}

// PersistCollection runs the import end-to-end:
//  1. Create the root folder (auto-rename if the user already has a root with this name).
//  2. Walk the tree; for each folder create it, for each request save a `SavedRequestFull`.
//     Within one folder, duplicate sibling names are disambiguated with " (n)" automatically.
//  3. Optionally create an Environment named after the collection, seeded with Variables.
//
// On partial failure we stop and surface the error — folders already created stay (they live in
// a freshly-named root so the user can delete the whole root to retry cleanly). We intentionally
// don't wrap the whole import in a single SQLite transaction: the existing repositories don't
// expose a tx-aware API, and the folder/request creation already uses smaller transactions under
// the hood. Keeping the blast radius to "one root folder" is acceptable for an import.
func (u *importUsecaseImpl) PersistCollection(
	ctx context.Context,
	col *entity.ImportedCollection,
	opts entity.ImportOptions,
) (*entity.ImportResult, error) {
	if col == nil || len(col.RootItems) == 0 {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportEmptyTree, nil)
	}

	rootName, err := u.pickUniqueRootFolderName(ctx, strings.TrimSpace(col.Name))
	if err != nil {
		return nil, err
	}
	logger.L().InfoContext(ctx, "import: create root folder", "name", rootName, "format", col.FormatLabel)

	rootID, err := u.folders.Create(ctx, &entity.FolderItem{
		Name:        rootName,
		Description: col.Description,
	})
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportPersistFailed, err)
	}

	result := &entity.ImportResult{
		RootFolderID:   rootID,
		RootFolderName: rootName,
		FormatLabel:    col.FormatLabel,
		Warnings:       col.Warnings,
	}
	if err := u.persistItems(ctx, rootID, col.RootItems, result); err != nil {
		return result, err
	}

	if opts.CreateEnvironment && len(col.Variables) > 0 {
		envID, envName, err := u.createEnvironmentForCollection(ctx, col, opts.ActivateEnvironment)
		if err != nil {
			// Keep the folder import — surface env error as warning so user can retry manually.
			result.Warnings = append(result.Warnings, "Could not create environment: "+err.Error())
		} else {
			result.EnvironmentID = envID
			result.EnvironmentName = envName
		}
	}
	return result, nil
}

// pickUniqueRootFolderName ensures we don't collide with ErrFolderRootNameConflict.
// Tries "Name", then "Name (2)", up to 100 attempts before giving up.
func (u *importUsecaseImpl) pickUniqueRootFolderName(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "Imported collection"
	}
	for n := 1; n <= 100; n++ {
		candidate := base
		if n > 1 {
			candidate = fmt.Sprintf("%s (%d)", base, n)
		}
		taken, err := u.folders.RootNameTaken(ctx, candidate, nil)
		if err != nil {
			return "", apperror.NewWithErrorDetail(constant.ErrDatabase, err)
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", apperror.NewWithErrorDetail(
		constant.ErrImportPersistFailed,
		fmt.Errorf("could not find a free root folder name for %q", base),
	)
}

func (u *importUsecaseImpl) persistItems(
	ctx context.Context,
	parentID string,
	items []entity.ImportedItem,
	result *entity.ImportResult,
) error {
	folderNamesUsed := make(map[string]struct{})
	requestNamesUsed := make(map[string]struct{})
	for _, it := range items {
		switch {
		case it.Folder != nil:
			name, err := pickUniqueSiblingName(it.Folder.Name, folderNamesUsed, func(candidate string) (bool, error) {
				return u.folders.ChildNameTaken(ctx, parentID, candidate, nil)
			})
			if err != nil {
				return apperror.NewWithErrorDetail(constant.ErrImportPersistFailed, err)
			}
			pid := parentID
			folderID, err := u.folders.Create(ctx, &entity.FolderItem{
				ParentID:    &pid,
				Name:        name,
				Description: it.Folder.Description,
			})
			if err != nil {
				return apperror.NewWithErrorDetail(constant.ErrImportPersistFailed, err)
			}
			result.FoldersCreated++
			if err := u.persistItems(ctx, folderID, it.Folder.Items, result); err != nil {
				return err
			}
		case it.Request != nil:
			name, err := pickUniqueSiblingName(it.Request.Name, requestNamesUsed, func(candidate string) (bool, error) {
				return u.reqs.ExistsNameInFolder(ctx, parentID, candidate, nil)
			})
			if err != nil {
				return apperror.NewWithErrorDetail(constant.ErrImportPersistFailed, err)
			}
			full := importedRequestToSaved(parentID, name, it.Request)
			normalizeRequestPayload(full)
			if _, err := u.reqs.CreateFull(ctx, full); err != nil {
				return apperror.NewWithErrorDetail(constant.ErrImportPersistFailed, err)
			}
			result.RequestsCreated++
		}
	}
	return nil
}

// pickUniqueSiblingName returns a name that neither collides with an already-minted sibling in
// this batch (`used`) nor with an existing DB row (reported by `taken`). On success, the chosen
// name is recorded in `used` so subsequent siblings don't reuse it.
//
// Tries "base", then "base (2)", …, up to 200 attempts before giving up with an error.
func pickUniqueSiblingName(base string, used map[string]struct{}, taken func(string) (bool, error)) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Untitled"
	}
	for n := 1; n <= 200; n++ {
		candidate := base
		if n > 1 {
			candidate = fmt.Sprintf("%s (%d)", base, n)
		}
		if _, already := used[candidate]; already {
			continue
		}
		dup, err := taken(candidate)
		if err != nil {
			return "", err
		}
		if !dup {
			used[candidate] = struct{}{}
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find a unique sibling name for %q", base)
}

func importedRequestToSaved(folderID, name string, r *entity.ImportedRequest) *entity.SavedRequestFull {
	return &entity.SavedRequestFull{
		FolderID:       folderID,
		Name:           name,
		Method:         r.Method,
		URL:            r.URL,
		BodyMode:       r.BodyMode,
		RawBody:        r.RawBody,
		Headers:        r.Headers,
		QueryParams:    r.QueryParams,
		FormFields:     r.FormFields,
		MultipartParts: r.MultipartParts,
		Auth:           r.Auth,
	}
}

func (u *importUsecaseImpl) createEnvironmentForCollection(
	ctx context.Context,
	col *entity.ImportedCollection,
	activate bool,
) (string, string, error) {
	baseName := strings.TrimSpace(col.Name)
	if baseName == "" {
		baseName = "Imported environment"
	}

	// Resolve a free environment name with the same "(n)" suffix pattern.
	var name string
	for n := 1; n <= 100; n++ {
		candidate := baseName
		if n > 1 {
			candidate = fmt.Sprintf("%s (%d)", baseName, n)
		}
		taken, err := u.envs.NameTaken(ctx, candidate, nil)
		if err != nil {
			return "", "", err
		}
		if !taken {
			name = candidate
			break
		}
	}
	if name == "" {
		return "", "", fmt.Errorf("could not find a free environment name for %q", baseName)
	}

	full, err := u.envs.Create(ctx, name, "Imported from "+col.FormatLabel)
	if err != nil {
		return "", "", err
	}

	rows := make([]entity.EnvVariableInput, 0, len(col.Variables))
	seen := make(map[string]struct{}, len(col.Variables))
	for i, v := range col.Variables {
		key := strings.TrimSpace(v.Key)
		if key == "" {
			continue
		}
		kl := strings.ToLower(key)
		if _, dup := seen[kl]; dup {
			continue
		}
		seen[kl] = struct{}{}
		rows = append(rows, entity.EnvVariableInput{
			Key:       key,
			Value:     v.Value,
			Enabled:   true,
			SortOrder: i,
		})
	}
	if len(rows) > 0 {
		if err := u.envs.SaveVariables(ctx, full.ID, rows); err != nil {
			return full.ID, full.Name, err
		}
	}
	if activate {
		if err := u.envs.SetActive(ctx, full.ID); err != nil {
			return full.ID, full.Name, err
		}
	}
	return full.ID, full.Name, nil
}
