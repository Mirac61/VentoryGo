package auth

import "errors"

var (
	ErrMismatchedPassword = errors.New("auth: password does not match hash")
	ErrInvalidHash        = errors.New("auth: hash is not in PHC format")
	ErrEmailTaken         = errors.New("auth: email already registered")
)
