package httperror

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Code }

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

func WithFields(base *Error, fields map[string]string) *Error {
	e := *base
	e.Fields = fields
	return &e
}

// ErrUnauthenticated is generic
var ErrUnauthenticated = &Error{
	Status:  http.StatusUnauthorized,
	Code:    "UNAUTHENTICATED",
	Message: "authentication required",
}
