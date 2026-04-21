package entity

// ProxySettings is the persisted proxy configuration (Phase 6).
type ProxySettings struct {
	Mode     string `json:"mode"`      // none | system | manual
	URL      string `json:"url"`       // manual: proxy base URL (scheme://host:port[/path])
	Username string `json:"username"`  // optional basic auth username
	Password string `json:"password"`  // optional — masked in UI; empty means "unchanged" on save
	NoProxy  string `json:"no_proxy"`  // comma-separated hosts / .suffix rules
}

// TrustedCASummary is one imported CA row for the Settings UI.
type TrustedCASummary struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

// ProxyTestResult is returned by SettingsHandler.TestProxy.
type ProxyTestResult struct {
	OK           bool   `json:"ok"`
	StatusCode   int    `json:"status_code"`
	DurationMs   int64  `json:"duration_ms"`
	ErrorMessage string `json:"error_message,omitempty"`
	FinalURL     string `json:"final_url,omitempty"`
}
