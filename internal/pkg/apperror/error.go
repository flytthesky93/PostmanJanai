package apperror

import (
	"fmt"
)

// AppError là cấu trúc lỗi tùy chỉnh
type AppError struct {
	Code    string // Mã lỗi (ví dụ: "DB_INSERT_FAILED")
	Message string // Thông báo lỗi cho người dùng
	Err     error  // Lỗi gốc (Root cause) để debug
}

type ErrDetail struct {
	Code    string
	Message string
}

// Hàm Error() giúp AppError thỏa mãn interface error của Go
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap cho phép sử dụng các hàm errors.Is và errors.As của Go
func (e *AppError) Unwrap() error {
	return e.Err
}

// New tạo một AppError mới
func New(code string, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewWithErrorDetail lấy dữ liệu từ ErrDetail và lỗi gốc (err) để tạo ra AppError
func NewWithErrorDetail(detail ErrDetail, err error) *AppError {
	return &AppError{
		Code:    detail.Code,
		Message: detail.Message,
		Err:     err,
	}
}
