package httperror

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	return c, rec
}

func TestWriteErrorUnknownErrorBecomes500(t *testing.T) {
	c, rec := setupContext()

	WriteError(c, errors.New("datenbank weg"))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"INTERNAL"`)
	assert.NotContains(t, rec.Body.String(), "datenbank weg")
}

func TestWriteErrorKnownErrorKeepsStatusAndCode(t *testing.T) {
	c, rec := setupContext()

	WriteError(c, &Error{Status: http.StatusNotFound, Code: "INVOICE_NOT_FOUND", Message: "invoice not found"})

	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"INVOICE_NOT_FOUND"`)
	assert.Contains(t, rec.Body.String(), `"message":"invoice not found"`)
}

func TestWriteErrorWrappedErrorKeepsStatusAndCode(t *testing.T) {
	c, rec := setupContext()

	WriteError(c, &Error{Status: http.StatusConflict, Code: "INVOICE_NOT_EDITABLE", Message: "only drafts can be edited"})

	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"INVOICE_NOT_EDITABLE"`)
}

func TestWriteErrorInvalidStatusFallsBackTo500(t *testing.T) {
	c, rec := setupContext()

	WriteError(c, &Error{Status: 0, Code: "FORGOTTEN_STATUS", Message: "oops"})

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"INTERNAL"`)
}

func TestWriteErrorIncludesRequestID(t *testing.T) {
	c, rec := setupContext()
	c.Set("request_id", "req-123")

	WriteError(c, &Error{Status: http.StatusNotFound, Code: "X", Message: "y"})

	assert.Contains(t, rec.Body.String(), `"requestId":"req-123"`)
}

func TestIsMatchesByCodeAcrossCopies(t *testing.T) {
	base := &Error{Status: http.StatusBadRequest, Code: "X", Message: "y"}

	assert.True(t, errors.Is(WithFields(base, map[string]string{"a": "b"}), base))
}

func TestBindValidationErrorsBecome422WithFields(t *testing.T) {
	c, rec := setupContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"keine-mail","password":"kurz"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	req := struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}{}
	ok := Bind(c, &req)

	assert.False(t, ok)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"VALIDATION_FAILED"`)
	assert.Contains(t, rec.Body.String(), `"fields"`)
	assert.True(t, strings.Contains(rec.Body.String(), `"email"`))
}

func TestBindBrokenJSONBecomes400(t *testing.T) {
	c, rec := setupContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":`))
	c.Request.Header.Set("Content-Type", "application/json")
	var req struct{ Email string }
	ok := Bind(c, &req)

	assert.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"INVALID_BODY"`)
}

func TestBindValidJSONReturnsTrue(t *testing.T) {
	c, _ := setupContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.de"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	ok := Bind(c, &req)

	assert.True(t, ok)
	assert.Equal(t, "a@b.de", req.Email)
}