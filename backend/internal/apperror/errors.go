package apperror

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"basepro/backend/internal/logging"
	"github.com/gin-gonic/gin"
)

const (
	CodeAuthUnauthorized = "AUTH_UNAUTHORIZED"
	CodeAuthForbidden    = "AUTH_FORBIDDEN"
	CodeAuthExpired      = "AUTH_EXPIRED"
	CodeAuthRefreshReuse = "AUTH_REFRESH_REUSED"
	CodeAuthRefreshBad   = "AUTH_REFRESH_INVALID"
	CodeValidationFailed = "VALIDATION_ERROR"
	CodeRateLimited      = "RATE_LIMITED"
)

type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Details    map[string]any
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func Write(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Details == nil {
			appErr.Details = map[string]any{}
		}
		logging.ForContext(c.Request.Context()).Warn("request_failed",
			slog.String("request_id", requestIDFromContext(c)),
			slog.String("code", appErr.Code),
			slog.String("message", appErr.Message),
			slog.Int("status", appErr.HTTPStatus),
		)
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": gin.H{
				"code":    appErr.Code,
				"message": appErr.Message,
				"details": appErr.Details,
			},
		})
		return
	}

	logging.ForContext(c.Request.Context()).Error("request_failed",
		slog.String("request_id", requestIDFromContext(c)),
		slog.String("code", "INTERNAL_ERROR"),
		slog.Int("status", http.StatusInternalServerError),
		slog.String("error_type", fmt.Sprintf("%T", err)),
	)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Internal server error",
			"details": map[string]any{},
		},
	})
}

func Unauthorized(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthUnauthorized, Message: message, Details: map[string]any{}}
}

func Forbidden(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusForbidden, Code: CodeAuthForbidden, Message: message, Details: map[string]any{}}
}

func Expired(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthExpired, Message: message, Details: map[string]any{}}
}

func RefreshReused(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthRefreshReuse, Message: message, Details: map[string]any{}}
}

func RefreshInvalid(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthRefreshBad, Message: message, Details: map[string]any{}}
}

func Validation(message string) *AppError {
	return ValidationWithDetails(message, map[string]any{"reason": message})
}

func ValidationWithDetails(message string, details map[string]any) *AppError {
	if details == nil {
		details = map[string]any{}
	}
	return &AppError{HTTPStatus: http.StatusBadRequest, Code: CodeValidationFailed, Message: message, Details: details}
}

func RateLimited(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusTooManyRequests, Code: CodeRateLimited, Message: message, Details: map[string]any{}}
}

func requestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get("request_id"); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return logging.RequestIDFromContext(c.Request.Context())
}
