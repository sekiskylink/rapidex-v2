package apperror

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeAuthUnauthorized = "AUTH_UNAUTHORIZED"
	CodeAuthForbidden    = "AUTH_FORBIDDEN"
	CodeAuthExpired      = "AUTH_EXPIRED"
	CodeAuthRefreshReuse = "AUTH_REFRESH_REUSED"
	CodeAuthRefreshBad   = "AUTH_REFRESH_INVALID"
)

type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func Write(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": gin.H{
				"code":    appErr.Code,
				"message": appErr.Message,
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Internal server error",
		},
	})
}

func Unauthorized(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthUnauthorized, Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusForbidden, Code: CodeAuthForbidden, Message: message}
}

func Expired(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthExpired, Message: message}
}

func RefreshReused(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthRefreshReuse, Message: message}
}

func RefreshInvalid(message string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: CodeAuthRefreshBad, Message: message}
}
