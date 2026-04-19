package entity

// RequestAuth is optional auth merged into headers/query after env substitution.
// Type: none | bearer | basic | apikey (case-insensitive when applied).
type RequestAuth struct {
	Type        string `json:"type,omitempty"`
	BearerToken string `json:"bearer_token,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	APIKeyName  string `json:"api_key_name,omitempty"`
	// APIKeyIn: "header" | "query"
	APIKeyIn string `json:"api_key_in,omitempty"`
}
