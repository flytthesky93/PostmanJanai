package entity

import "time"

type HistoryItem struct {
	ID                   string    `json:"id"`
	Method               string    `json:"method"`
	URL                  string    `json:"url"`
	StatusCode           int       `json:"status_code"`
	DurationMs           *int      `json:"duration_ms,omitempty"`
	ResponseSizeBytes    *int      `json:"response_size_bytes,omitempty"`
	RequestHeadersJSON   *string   `json:"request_headers_json,omitempty"`
	ResponseHeadersJSON  *string   `json:"response_headers_json,omitempty"`
	RequestBody          *string   `json:"request_body,omitempty"`
	ResponseBody         *string   `json:"response_body,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}
