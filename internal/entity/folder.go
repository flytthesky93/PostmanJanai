package entity

import "time"

// FolderItem is a folder node (root or nested). Wails JSON.
type FolderItem struct {
	ID          string    `json:"id"`
	ParentID    *string   `json:"parent_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateFolderInput is the payload for creating a root or nested folder.
type CreateFolderInput struct {
	ParentID    *string `json:"parent_id,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
}
