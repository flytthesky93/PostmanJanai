package entity

import "time"

// EnvironmentSummary is a row for sidebar / dropdown lists.
type EnvironmentSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EnvironmentVariableRow is one key/value row for an environment.
type EnvironmentVariableRow struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Kind      string `json:"kind"` // "plain" | "secret"
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}

// EnvironmentFull is metadata + variables for the editor panel.
type EnvironmentFull struct {
	EnvironmentSummary
	Variables []EnvironmentVariableRow `json:"variables"`
}

// EnvVariableInput is sent from the UI when saving variables (full replace).
type EnvVariableInput struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Kind      string `json:"kind"` // "plain" | "secret" — empty means plain
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}
