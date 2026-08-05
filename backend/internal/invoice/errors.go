package invoice

import "errors"

var (
	ErrNotFound          = errors.New("invoice not found")
	ErrNotDeletable      = errors.New("invoice isn't deletable")
	ErrNotUpdatable      = errors.New("invoice not updatable")
	ErrInvalidInput      = errors.New("invalid invoice data")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrMissingOwner      = errors.New("owner id missing from context")
)
