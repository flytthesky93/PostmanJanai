package constant

import "PostmanJanai/internal/pkg/apperror"

var (
	// Nhóm lỗi Hệ thống
	ErrInternal = apperror.ErrDetail{Code: "SYS_001", Message: "Lỗi hệ thống nội bộ."}
	ErrDatabase = apperror.ErrDetail{Code: "SYS_002", Message: "Lỗi truy vấn cơ sở dữ liệu."}

	// Nhóm lỗi Request
	ErrInvalidURL     = apperror.ErrDetail{Code: "REQ_101", Message: "Địa chỉ URL không hợp lệ."}
	ErrRequestTimeout = apperror.ErrDetail{Code: "REQ_102", Message: "Yêu cầu hết thời gian phản hồi."}

	// Nhóm lỗi History
	ErrHistoryNotFound = apperror.ErrDetail{Code: "HIS_201", Message: "Không tìm thấy lịch sử yêu cầu."}

	// Nhóm lỗi Workspace
	ErrWorkspaceAlreadyExisted = apperror.ErrDetail{Code: "WS_301", Message: "Workspace đã tồn tại"}
	ErrWorkspaceSaveFail       = apperror.ErrDetail{Code: "WS_302", Message: "Tạo workspace mới thất bại"}
)
