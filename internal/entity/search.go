package entity

// SearchFolderHit is a folder that matched a search query. `Path` is the
// list of ancestor folder names from the root (inclusive) down to — but NOT
// including — this folder itself. For a root folder, Path is empty.
// AncestorIDs is the id chain from root to this folder INCLUSIVE — lets the
// frontend expand each level to reveal the hit.
type SearchFolderHit struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ParentID    *string  `json:"parent_id,omitempty"`
	RootID      string   `json:"root_id"`
	Path        []string `json:"path"`
	AncestorIDs []string `json:"ancestor_ids"`
	Description string   `json:"description,omitempty"`
}

// SearchRequestHit is a saved request that matched a search query. `Path`
// is the list of ancestor folder names from the root (inclusive) down to
// the folder that directly holds this request. AncestorIDs is the id chain
// from root down to (and including) the folder containing this request.
type SearchRequestHit struct {
	ID          string   `json:"id"`
	FolderID    string   `json:"folder_id"`
	RootID      string   `json:"root_id"`
	Name        string   `json:"name"`
	Method      string   `json:"method"`
	URL         string   `json:"url"`
	Path        []string `json:"path"`
	AncestorIDs []string `json:"ancestor_ids"`
}

// SearchResults aggregates hits across folders + saved requests for the
// sidebar's unified search UX.
type SearchResults struct {
	Query    string              `json:"query"`
	Folders  []*SearchFolderHit  `json:"folders"`
	Requests []*SearchRequestHit `json:"requests"`
	// Truncated is true if either collection hit the `limit` cap.
	Truncated bool `json:"truncated"`
}
