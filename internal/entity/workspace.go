package entity

import "time"

type WorkspaceItem struct {
	ID                   int       `json:"id"`
	WorkspaceName        string    `json:"workspace_name"`
	WorkspaceDescription string    `json:"workspace_description"`
	CreatedAt            time.Time `json:"created_at"`
}
