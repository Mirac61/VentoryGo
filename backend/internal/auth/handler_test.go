package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	created []User
	err     error
}

func (f *fakeRepo) Create(_ context.Context, user User) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, user)
	return nil
}

func postJSON(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	return c, rec
}

func TestRegisterRejectsInvalidFieldsWith422(t *testing.T) {
	h := NewHandler(nil)

	c, rec := postJSON(`{"email":"keine-mail","password":"kurz"}`)
	h.Register(c)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Equal(t, map[string]string{
		"email":    "email",
		"password": "min",
	}, body.Errors)
}

func TestRegisterRejectsBrokenJSONWith400(t *testing.T) {
	h := NewHandler(nil)

	c, rec := postJSON(`{"email":`)
	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterCreatesUserAndOmitsPasswordHash(t *testing.T) {
	repo := &fakeRepo{}
	h := NewHandler(NewService(repo))

	c, rec := postJSON(`{"email":"max@example.com","password":"correct horse battery"}`)
	h.Register(c)

	require.Equal(t, http.StatusCreated, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Equal(t, "max@example.com", body["email"])
	assert.NotEmpty(t, body["id"])
	assert.NotContains(t, body, "passwordHash")
	assert.NotContains(t, body, "password_hash")

	// Faengt den Hash auch dann, wenn er unter einem anderen Key auftauchen wuerde.
	assert.NotContains(t, rec.Body.String(), "argon2")

	require.Len(t, repo.created, 1)
	assert.True(t, strings.HasPrefix(repo.created[0].PasswordHash, "$argon2id$"))
}

func TestRegisterMapsEmailTakenTo409(t *testing.T) {
	h := NewHandler(NewService(&fakeRepo{err: ErrEmailTaken}))

	c, rec := postJSON(`{"email":"max@example.com","password":"correct horse battery"}`)
	h.Register(c)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.NotEqual(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestRegisterPasswordLengthBoundary(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		want     int
	}{
		{"elf Zeichen", strings.Repeat("a", 11), http.StatusUnprocessableEntity},
		{"zwoelf Zeichen", strings.Repeat("a", 12), http.StatusCreated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(NewService(&fakeRepo{}))

			c, rec := postJSON(fmt.Sprintf(`{"email":"max@example.com","password":%q}`, tc.password))
			h.Register(c)

			assert.Equal(t, tc.want, rec.Code)
		})
	}
}
