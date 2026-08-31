package invoice

import (
	"errors"
	"net/http"

	"github.com/Mirac61/VentoryGo/backend/internal/httperror"
)

var (
	ErrNotFound = &httperror.Error{
		Status:  http.StatusNotFound,
		Code:    "INVOICE_NOT_FOUND",
		Message: "invoice not found",
	}
	ErrNotDeletable = &httperror.Error{
		Status:  http.StatusConflict,
		Code:    "INVOICE_NOT_DELETABLE",
		Message: "only drafts can be deleted",
	}
	ErrNotUpdatable = &httperror.Error{
		Status:  http.StatusConflict,
		Code:    "INVOICE_NOT_EDITABLE",
		Message: "only drafts can be edited",
	}
	ErrInvalidInput = &httperror.Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    "INVOICE_INVALID",
		Message: "invalid invoice data",
	}
	ErrInvalidTransition = &httperror.Error{
		Status:  http.StatusConflict,
		Code:    "INVOICE_INVALID_TRANSITION",
		Message: "invalid status transition",
	}
	ErrIncomplete = &httperror.Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    "INVOICE_INCOMPLETE",
		Message: "required fields missing",
	}
)

// Both stay plain errors on purpose: they report a broken server, not a bad
// request. With no *httperror.Error to unwrap, WriteError logs them and falls
// into its 500 branch instead of telling the client what to fix.
var (
	ErrMissingOwner  = errors.New("owner id missing from context")
	ErrNoPDFRenderer = errors.New("pdf renderer not configured")
)
