package entity

// KeyValue is a key/value pair for headers and query (JSON for Wails).
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// BodyMode describes the request body kind.
type BodyMode string

const (
	BodyModeNone              BodyMode = "none"
	BodyModeRaw               BodyMode = "raw"
	BodyModeXML               BodyMode = "xml"
	BodyModeFormURLEncoded    BodyMode = "form_urlencoded"
	BodyModeMultipartFormData BodyMode = "multipart"
)

// MultipartPart is one part of multipart/form-data (text field or local file).
type MultipartPart struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"` // "text" | "file"
	Value    string `json:"value,omitempty"`
	FilePath string `json:"file_path,omitempty"`
}

// HTTPExecuteInput is the payload from the UI when Send is pressed (ad-hoc request, not saved).
type HTTPExecuteInput struct {
	Method      string     `json:"method"`
	URL         string     `json:"url"`
	Headers     []KeyValue `json:"headers,omitempty"`
	QueryParams []KeyValue `json:"query_params,omitempty"`

	// Optional UUIDs for history: root folder (sidebar selection) and/or saved request.
	RootFolderID *string `json:"root_folder_id,omitempty"`
	RequestID    *string `json:"request_id,omitempty"`

	BodyMode string `json:"body_mode,omitempty"` // BodyMode* (string for JSON)

	// Raw (JSON/text) when body_mode is raw or default.
	Body string `json:"body,omitempty"`

	// application/x-www-form-urlencoded — key/value pairs.
	FormFields []KeyValue `json:"form_fields,omitempty"`

	// multipart/form-data — order preserved.
	MultipartParts []MultipartPart `json:"multipart_parts,omitempty"`

	// Optional auth (Bearer / Basic / API Key); merged after {{var}} resolution.
	Auth *RequestAuth `json:"auth,omitempty"`

	// InsecureSkipVerify disables TLS certificate verification for this send only
	// (mirrors saved request flag when executing a saved request).
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

// HTTPExecuteResult is the outcome of HTTP execution (HTTP response or network/timeout error).
type HTTPExecuteResult struct {
	StatusCode        int        `json:"status_code"`
	DurationMs        int64      `json:"duration_ms"`
	ResponseSizeBytes int64      `json:"response_size_bytes"`
	ResponseHeaders   []KeyValue `json:"response_headers,omitempty"`
	ResponseBody      string     `json:"response_body"`
	BodyTruncated     bool       `json:"body_truncated"`
	ErrorMessage      string     `json:"error_message,omitempty"`

	// FinalURL is the URL after merging query params (scheme + host + path + query).
	FinalURL string `json:"final_url,omitempty"`

	// Phase 8 — Post-response capture + assertion outcomes (saved request only).
	Captures   []CaptureResult   `json:"captures,omitempty"`
	Assertions []AssertionResult `json:"assertions,omitempty"`

	// Snapshots for persisting history (not serialized to the frontend).
	RequestHeadersSnapshot []KeyValue `json:"-"`
	RequestBodySnapshot    string     `json:"-"`
}
