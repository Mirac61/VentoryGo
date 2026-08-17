package auth

import (
	"errors"
	"net/http"

	"github.com/Mirac61/VentoryGo/backend/internal/httperror"
)

var (
	ErrEmailTaken = &httperror.Error{
		Status:  http.StatusConflict,
		Code:    "EMAIL_TAKEN",
		Message: "email already registered",
	}
	ErrInvalidCredentials = &httperror.Error{
		Status:  http.StatusUnauthorized,
		Code:    "INVALID_CREDENTIALS",
		Message: "invalid email or password",
	}
	ErrSessionNotFound = &httperror.Error{
		Status:  http.StatusUnauthorized,
		Code:    "SESSION_EXPIRED",
		Message: "session expired or invalid",
	}
	ErrUserNotFound = &httperror.Error{
		Status:  http.StatusNotFound,
		Code:    "USER_NOT_FOUND",
		Message: "user not found",
	}
)

// ErrMismatchedPassword and ErrInvalidHash stay plain errors: they are
// internal details of password.go and must never reach the client.
var (
	ErrMismatchedPassword = errors.New("auth: password does not match hash")
	ErrInvalidHash        = errors.New("auth: hash is not in PHC format")
)
