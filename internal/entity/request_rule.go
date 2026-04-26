package entity

// Capture rule sources / scopes — keep in sync with constant.CaptureSource* / CaptureScope*.

// RequestCaptureRow — Wails JSON for one capture rule attached to a saved request.
type RequestCaptureRow struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Source         string `json:"source"`
	Expression     string `json:"expression"`
	TargetScope    string `json:"target_scope"`
	TargetVariable string `json:"target_variable"`
	Enabled        bool   `json:"enabled"`
	SortOrder      int    `json:"sort_order"`
}

// RequestCaptureInput — payload from UI when saving (full replace).
type RequestCaptureInput struct {
	Name           string `json:"name"`
	Source         string `json:"source"`
	Expression     string `json:"expression"`
	TargetScope    string `json:"target_scope"`
	TargetVariable string `json:"target_variable"`
	Enabled        bool   `json:"enabled"`
	SortOrder      int    `json:"sort_order"`
}

// RequestAssertionRow — Wails JSON for one assertion attached to a saved request.
type RequestAssertionRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Expression string `json:"expression"`
	Operator   string `json:"operator"`
	Expected   string `json:"expected"`
	Enabled    bool   `json:"enabled"`
	SortOrder  int    `json:"sort_order"`
}

// RequestAssertionInput — payload from UI when saving (full replace).
type RequestAssertionInput struct {
	Name       string `json:"name"`
	Source     string `json:"source"`
	Expression string `json:"expression"`
	Operator   string `json:"operator"`
	Expected   string `json:"expected"`
	Enabled    bool   `json:"enabled"`
	SortOrder  int    `json:"sort_order"`
}

// CaptureResult — outcome of one capture rule against one response.
type CaptureResult struct {
	Name           string `json:"name"`
	TargetScope    string `json:"target_scope"`
	TargetVariable string `json:"target_variable"`
	Source         string `json:"source"`
	Expression     string `json:"expression"`
	Value          string `json:"value"`
	Captured       bool   `json:"captured"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// AssertionResult — outcome of one assertion rule against one response.
type AssertionResult struct {
	Name         string `json:"name"`
	Source       string `json:"source"`
	Expression   string `json:"expression"`
	Operator     string `json:"operator"`
	Expected     string `json:"expected"`
	Actual       string `json:"actual"`
	Passed       bool   `json:"passed"`
	ErrorMessage string `json:"error_message,omitempty"`
}
