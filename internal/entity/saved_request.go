package entity

import "time"

// SavedRequestSummary is a lightweight row for tree lists (no headers/body).
type SavedRequestSummary struct {
	ID        string    `json:"id"`
	FolderID  string    `json:"folder_id"`
	Name      string    `json:"name"`
	Method    string    `json:"method"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SavedRequestFull is a persisted request with all parts (Wails JSON).
type SavedRequestFull struct {
	ID             string          `json:"id"`
	FolderID       string          `json:"folder_id"`
	Name           string          `json:"name"`
	Method         string          `json:"method"`
	URL            string          `json:"url"`
	BodyMode       string          `json:"body_mode"`
	RawBody        *string         `json:"raw_body,omitempty"`
	Headers        []KeyValue      `json:"headers,omitempty"`
	QueryParams    []KeyValue      `json:"query_params,omitempty"`
	FormFields     []KeyValue      `json:"form_fields,omitempty"`
	MultipartParts []MultipartPart `json:"multipart_parts,omitempty"`
	Auth           *RequestAuth    `json:"auth,omitempty"`
	InsecureSkipVerify bool        `json:"insecure_skip_verify,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
