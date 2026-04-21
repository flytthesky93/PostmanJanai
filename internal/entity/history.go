package entity

import "time"

// HistorySummary is a lightweight row for the sidebar list (no request/response bodies).
type HistorySummary struct {
	ID                string    `json:"id"`
	RootFolderID      *string   `json:"root_folder_id,omitempty"`
	RequestID         *string   `json:"request_id,omitempty"`
	InsecureTLS       bool      `json:"insecure_tls,omitempty"`
	Method            string    `json:"method"`
	URL               string    `json:"url"`
	StatusCode        int       `json:"status_code"`
	DurationMs        *int      `json:"duration_ms,omitempty"`
	ResponseSizeBytes *int      `json:"response_size_bytes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type HistoryItem struct {
	ID                  string    `json:"id"`
	RootFolderID        *string   `json:"root_folder_id,omitempty"`
	RequestID           *string   `json:"request_id,omitempty"`
	InsecureTLS         bool      `json:"insecure_tls,omitempty"`
	Method              string    `json:"method"`
	URL                 string    `json:"url"`
	StatusCode          int       `json:"status_code"`
	DurationMs          *int      `json:"duration_ms,omitempty"`
	ResponseSizeBytes   *int      `json:"response_size_bytes,omitempty"`
	RequestHeadersJSON  *string   `json:"request_headers_json,omitempty"`
	ResponseHeadersJSON *string   `json:"response_headers_json,omitempty"`
	RequestBody         *string   `json:"request_body,omitempty"`
	ResponseBody        *string   `json:"response_body,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}
