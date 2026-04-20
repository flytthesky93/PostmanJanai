package constant

import "PostmanJanai/internal/pkg/apperror"

var (
	// System / DB
	ErrInternal = apperror.ErrDetail{Code: "SYS_001", Message: "Internal server error."}
	ErrDatabase = apperror.ErrDetail{Code: "SYS_002", Message: "Database error."}

	// HTTP / request
	ErrInvalidURL     = apperror.ErrDetail{Code: "REQ_101", Message: "Invalid URL."}
	ErrRequestTimeout = apperror.ErrDetail{Code: "REQ_102", Message: "Request timed out."}
	ErrInvalidCurl    = apperror.ErrDetail{Code: "REQ_103", Message: "Could not parse cURL command."}

	// History
	ErrHistoryNotFound = apperror.ErrDetail{Code: "HIS_201", Message: "Request history not found."}

	// Folder (root + nested; replaces workspace/collection)
	ErrFolderRootNameConflict  = apperror.ErrDetail{Code: "FOL_301", Message: "A root folder with this name already exists"}
	ErrFolderChildNameConflict = apperror.ErrDetail{Code: "FOL_302", Message: "A folder with this name already exists here"}
	ErrFolderNotFound          = apperror.ErrDetail{Code: "FOL_303", Message: "Folder not found"}
	ErrFolderSaveFail          = apperror.ErrDetail{Code: "FOL_304", Message: "Failed to save folder"}

	// Saved request
	ErrSavedRequestNotFound     = apperror.ErrDetail{Code: "REQ_501", Message: "Request not found"}
	ErrSavedRequestNameConflict = apperror.ErrDetail{Code: "REQ_502", Message: "A request with this name already exists in this location"}

	// Environment (global app)
	ErrEnvironmentNotFound      = apperror.ErrDetail{Code: "ENV_601", Message: "Environment not found"}
	ErrEnvironmentNameConflict = apperror.ErrDetail{Code: "ENV_602", Message: "An environment with this name already exists"}
	ErrEnvironmentDuplicateVariableKey = apperror.ErrDetail{Code: "ENV_603", Message: "Duplicate variable key in the same environment"}

	// Import collection
	ErrImportFileOpen       = apperror.ErrDetail{Code: "IMP_701", Message: "Could not open the selected file"}
	ErrImportFileTooLarge   = apperror.ErrDetail{Code: "IMP_702", Message: "Collection file exceeds the maximum allowed size"}
	ErrImportFileEmpty      = apperror.ErrDetail{Code: "IMP_703", Message: "Collection file is empty"}
	ErrImportFormatUnknown  = apperror.ErrDetail{Code: "IMP_704", Message: "Unsupported or unrecognized collection format"}
	ErrImportParseFailed    = apperror.ErrDetail{Code: "IMP_705", Message: "Failed to parse collection"}
	ErrImportEmptyTree      = apperror.ErrDetail{Code: "IMP_706", Message: "The collection does not contain any requests"}
	ErrImportPersistFailed  = apperror.ErrDetail{Code: "IMP_707", Message: "Failed to save the imported collection"}
)
