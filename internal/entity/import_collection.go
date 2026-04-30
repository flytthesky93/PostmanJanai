package entity

// ImportedCollection is the parsed tree produced by any of the supported format importers
// (Postman v2.1 / v2.0, OpenAPI 3.x, Insomnia v4). It's a format-agnostic shape consumed by
// the import usecase to persist folders + saved requests (and optionally one environment).
//
// The JSON tags make it visible to the Wails binding (UI preview) without coupling UI to
// any specific source format.
type ImportedCollection struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Variables   []ImportedVariable   `json:"variables,omitempty"`
	RootItems   []ImportedItem       `json:"root_items,omitempty"`
	FormatLabel string               `json:"format_label"`
	Warnings    []string             `json:"warnings,omitempty"`
}

// ImportedVariable is one entry for an Environment derived from the source file.
type ImportedVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ImportedItem is either a folder or a request node. Exactly one of Folder/Request is non-nil.
type ImportedItem struct {
	Folder  *ImportedFolder  `json:"folder,omitempty"`
	Request *ImportedRequest `json:"request,omitempty"`
}

// ImportedFolder groups nested items (folders + requests) under a named node.
type ImportedFolder struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Items       []ImportedItem `json:"items,omitempty"`
}

// ImportedRequest is the format-agnostic request payload (same shape as SavedRequestFull minus IDs).
type ImportedRequest struct {
	Name           string          `json:"name"`
	Method         string          `json:"method"`
	URL            string          `json:"url"`
	BodyMode       string          `json:"body_mode,omitempty"`
	RawBody        *string         `json:"raw_body,omitempty"`
	Headers        []KeyValue      `json:"headers,omitempty"`
	QueryParams    []KeyValue      `json:"query_params,omitempty"`
	FormFields     []KeyValue      `json:"form_fields,omitempty"`
	MultipartParts []MultipartPart `json:"multipart_parts,omitempty"`
	Auth           *RequestAuth    `json:"auth,omitempty"`
	// Phase 9 — optional scripts from imported collections (e.g. Postman events).
	PreRequestScript   string `json:"pre_request_script,omitempty"`
	PostResponseScript string `json:"post_response_script,omitempty"`
}

// ImportOptions controls side-effects of persisting an ImportedCollection.
type ImportOptions struct {
	// CreateEnvironment creates an Environment row named after the collection and seeds
	// it with Variables. Already-existing names are disambiguated with " (2)", " (3)", …
	CreateEnvironment bool `json:"create_environment"`

	// ActivateEnvironment sets the newly-created environment as active when true.
	ActivateEnvironment bool `json:"activate_environment"`
}

// ImportResult summarises what was created so the UI can navigate / toast accordingly.
type ImportResult struct {
	RootFolderID    string   `json:"root_folder_id"`
	RootFolderName  string   `json:"root_folder_name"`
	FoldersCreated  int      `json:"folders_created"`
	RequestsCreated int      `json:"requests_created"`
	EnvironmentID   string   `json:"environment_id,omitempty"`
	EnvironmentName string   `json:"environment_name,omitempty"`
	FormatLabel     string   `json:"format_label"`
	Warnings        []string `json:"warnings,omitempty"`
}
