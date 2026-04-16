package constant

import "PostmanJanai/internal/pkg/apperror"

var (
	// System / DB
	ErrInternal = apperror.ErrDetail{Code: "SYS_001", Message: "Internal server error."}
	ErrDatabase = apperror.ErrDetail{Code: "SYS_002", Message: "Database error."}

	// HTTP / request
	ErrInvalidURL     = apperror.ErrDetail{Code: "REQ_101", Message: "Invalid URL."}
	ErrRequestTimeout = apperror.ErrDetail{Code: "REQ_102", Message: "Request timed out."}

	// History
	ErrHistoryNotFound = apperror.ErrDetail{Code: "HIS_201", Message: "Request history not found."}

	// Workspace
	ErrWorkspaceAlreadyExisted = apperror.ErrDetail{Code: "WS_301", Message: "Workspace already exists"}
	ErrWorkspaceSaveFail       = apperror.ErrDetail{Code: "WS_302", Message: "Failed to create workspace"}
)
