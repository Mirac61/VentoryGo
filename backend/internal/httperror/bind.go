package httperror

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var (
	errInvalidBody = &Error{Status: http.StatusBadRequest, Code: "INVALID_BODY", Message: "invalid request body"}
	errValidation  = &Error{Status: http.StatusUnprocessableEntity, Code: "VALIDATION_FAILED", Message: "invalid input"}
)

func Bind(c *gin.Context, req any) bool {
	err := c.ShouldBindJSON(req)
	if err == nil {
		return true
	}
	if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		fields := make(map[string]string, len(validationErrs))
		for _, fieldErr := range validationErrs {
			fields[strings.ToLower(fieldErr.Field())] = fieldErr.Tag()
		}
		WriteError(c, WithFields(errValidation, fields))
		return false
	}
	WriteError(c, errInvalidBody)
	return false
}
