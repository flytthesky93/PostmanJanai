package usecase

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/repository"
	"context"
	"strings"
)

// SearchUsecase powers the sidebar's global search (folders + saved requests).
// History is filtered client-side so it isn't exposed here.
type SearchUsecase interface {
	SearchTree(ctx context.Context, query string, limit int) (*entity.SearchResults, error)
}

type searchUsecaseImpl struct {
	folders  repository.FolderRepository
	requests repository.RequestRepository
}

func NewSearchUsecase(folders repository.FolderRepository, requests repository.RequestRepository) SearchUsecase {
	return &searchUsecaseImpl{folders: folders, requests: requests}
}

const (
	searchDefaultLimit = 100
	searchMaxLimit     = 500
)

func (u *searchUsecaseImpl) SearchTree(ctx context.Context, query string, limit int) (*entity.SearchResults, error) {
	q := strings.TrimSpace(query)
	out := &entity.SearchResults{Query: q, Folders: []*entity.SearchFolderHit{}, Requests: []*entity.SearchRequestHit{}}
	if q == "" {
		return out, nil
	}
	if limit <= 0 {
		limit = searchDefaultLimit
	}
	if limit > searchMaxLimit {
		limit = searchMaxLimit
	}

	// Single skeleton pull — cheap even with thousands of folders — lets us
	// compute breadcrumbs without recursive GetByID calls per hit.
	all, err := u.folders.ListAllSkeleton(ctx)
	if err != nil {
		return nil, err
	}
	index := buildFolderIndex(all)

	folderHits, fTrunc, err := u.folders.SearchByName(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	out.Folders = make([]*entity.SearchFolderHit, 0, len(folderHits))
	for _, f := range folderHits {
		path, ancestors, rootID := chainForFolder(index, f.ID)
		// `path` currently contains [root..f]; the UI wants ancestors only.
		if n := len(path); n > 0 {
			path = path[:n-1]
		}
		out.Folders = append(out.Folders, &entity.SearchFolderHit{
			ID:          f.ID,
			Name:        f.Name,
			ParentID:    f.ParentID,
			RootID:      rootID,
			Path:        path,
			AncestorIDs: ancestors,
			Description: f.Description,
		})
	}

	reqHits, rTrunc, err := u.requests.SearchByNameOrURL(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	out.Requests = make([]*entity.SearchRequestHit, 0, len(reqHits))
	for _, r := range reqHits {
		path, ancestors, rootID := chainForFolder(index, r.FolderID)
		out.Requests = append(out.Requests, &entity.SearchRequestHit{
			ID:          r.ID,
			FolderID:    r.FolderID,
			RootID:      rootID,
			Name:        r.Name,
			Method:      r.Method,
			URL:         r.URL,
			Path:        path,
			AncestorIDs: ancestors,
		})
	}

	out.Truncated = fTrunc || rTrunc
	return out, nil
}

type folderIndexEntry struct {
	name     string
	parentID string
}

func buildFolderIndex(all []*entity.FolderItem) map[string]folderIndexEntry {
	idx := make(map[string]folderIndexEntry, len(all))
	for _, f := range all {
		p := ""
		if f.ParentID != nil {
			p = *f.ParentID
		}
		idx[f.ID] = folderIndexEntry{name: f.Name, parentID: p}
	}
	return idx
}

// pathForFolder walks parent links to produce [root, ..., self] and returns
// the root folder id. Cycle-safe via visited set. Returns (nil, "") when the
// id is empty or the folder isn't in the index.
func pathForFolder(index map[string]folderIndexEntry, folderID string) ([]string, string) {
	names, _, root := chainForFolder(index, folderID)
	return names, root
}

// chainForFolder is the fuller variant used by SearchTree: returns both the
// name chain and the id chain (both ordered root → self), plus the resolved
// root id. Cycle-safe.
func chainForFolder(index map[string]folderIndexEntry, folderID string) ([]string, []string, string) {
	if folderID == "" {
		return nil, nil, ""
	}
	if _, ok := index[folderID]; !ok {
		return nil, nil, ""
	}
	var names []string
	var ids []string
	visited := make(map[string]struct{}, 4)
	rootID := folderID
	cur := folderID
	for {
		if _, seen := visited[cur]; seen {
			break
		}
		visited[cur] = struct{}{}
		entry, ok := index[cur]
		if !ok {
			break
		}
		names = append(names, entry.name)
		ids = append(ids, cur)
		rootID = cur
		if entry.parentID == "" {
			break
		}
		cur = entry.parentID
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
		ids[i], ids[j] = ids[j], ids[i]
	}
	return names, ids, rootID
}
