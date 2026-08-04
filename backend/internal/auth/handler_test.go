package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (f *fakeRepo) FindByEmail(_ context.Context, email string) (User, error) {
	for _, user := range f.created {
		if strings.EqualFold(user.Email, email) {
			return user, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (f *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (User, error) {
	for _, user := range f.created {
		if user.ID == id.String() {
			return user, nil
		}
	}
	return User{}, ErrUserNotFound
}

type fakeSessionStore struct {
	created        []Session
	expiredCleared []uuid.UUID
	err            error
	touched        int
}

func (f *fakeSessionStore) Create(_ context.Context, s Session) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, s)
	return nil
}

func (f *fakeSessionStore) index(tokenHash []byte) int {
	for i, s := range f.created {
		if bytes.Equal(s.TokenHash, tokenHash) {
			return i
		}
	}
	return -1
}

func (f *fakeSessionStore) remove(i int) {
	f.created = append(f.created[:i], f.created[i+1:]...)
}

func (f *fakeSessionStore) Get(_ context.Context, tokenHash []byte) (Session, error) {
	i := f.index(tokenHash)
	if i < 0 {
		return Session{}, ErrSessionNotFound
	}
	if !f.created[i].ExpiresAt.After(time.Now().UTC()) {
		f.remove(i)
		return Session{}, ErrSessionNotFound
	}
	return f.created[i], nil
}

func (f *fakeSessionStore) Touch(_ context.Context, tokenHash []byte, expiresAt time.Time) error {
	i := f.index(tokenHash)
	if i < 0 {
		return ErrSessionNotFound
	}
	f.created[i].ExpiresAt = expiresAt
	f.touched++
	return nil
}

func (f *fakeSessionStore) Delete(_ context.Context, tokenHash []byte) error {
	if f.err != nil {
		return f.err
	}
	if i := f.index(tokenHash); i >= 0 {
		f.remove(i)
	}
	return nil
}

func (f *fakeSessionStore) DeleteExpiredByUser(_ context.Context, userID uuid.UUID) error {
	f.expiredCleared = append(f.expiredCleared, userID)
	return nil
}

func serviceWithUser(t *testing.T, email, password string) (*Service, *fakeSessionStore) {
	t.Helper()
	store := &fakeSessionStore{}
	s := NewService(&fakeRepo{}, store)
	_, err := s.Register(context.Background(), email, password)
	require.NoError(t, err)
	return s, store
}

func serveLogin(h *Handler, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/auth/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "session" {
			return ck
		}
	}
	require.Fail(t, "Set-Cookie fuer 'session' fehlt")
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
	h := NewHandler(nil, true)

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
	h := NewHandler(nil, true)

	c, rec := postJSON(`{"email":`)
	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterCreatesUserAndOmitsPasswordHash(t *testing.T) {
	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, nil), true)

	c, rec := postJSON(`{"email":"max@example.com","password":"correct horse battery"}`)
	h.Register(c)

	require.Equal(t, http.StatusCreated, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Equal(t, "max@example.com", body["email"])
	assert.NotEmpty(t, body["id"])
	assert.NotContains(t, body, "passwordHash")
	assert.NotContains(t, body, "password_hash")

	assert.NotContains(t, rec.Body.String(), "argon2")

	require.Len(t, repo.created, 1)
	assert.True(t, strings.HasPrefix(repo.created[0].PasswordHash, "$argon2id$"))
}

func TestRegisterMapsEmailTakenTo409(t *testing.T) {
	h := NewHandler(NewService(&fakeRepo{err: ErrEmailTaken}, nil), true)

	c, rec := postJSON(`{"email":"max@example.com","password":"correct horse battery"}`)
	h.Register(c)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.NotEqual(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestLoginSetsSessionCookieAndKeepsTokenOutOfBody(t *testing.T) {
	service, store := serviceWithUser(t, "max@example.com", "correct horse battery")
	h := NewHandler(service, true)

	rec := serveLogin(h, `{"email":"max@example.com","password":"correct horse battery"}`)

	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String(), "204 traegt keinen Body")
	require.Len(t, store.created, 1)

	cookie := sessionCookie(t, rec)

	assert.NotEmpty(t, cookie.Value)
	assert.True(t, cookie.HttpOnly, "ohne HttpOnly kommt JS an das Token")
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, int(defaultSessionTTL.Seconds()), cookie.MaxAge)

	assert.NotContains(t, rec.Body.String(), cookie.Value)
}

func TestLoginCookieSecureFollowsConfig(t *testing.T) {
	for _, secure := range []bool{true, false} {
		t.Run(fmt.Sprintf("secure=%v", secure), func(t *testing.T) {
			service, _ := serviceWithUser(t, "max@example.com", "correct horse battery")
			h := NewHandler(service, secure)

			rec := serveLogin(h, `{"email":"max@example.com","password":"correct horse battery"}`)

			require.Equal(t, http.StatusNoContent, rec.Code)
			assert.Equal(t, secure, sessionCookie(t, rec).Secure)
		})
	}
}

func TestCookieSecureFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   string
		want    bool
		wantErr bool
	}{
		{name: "nicht gesetzt", want: true},
		{name: "false", value: "false", want: false},
		{name: "0", value: "0", want: false},
		{name: "true", value: "true", want: true},
		{name: "kaputt", value: "vielleicht", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("COOKIE_SECURE", tc.value)

			secure, err := CookieSecureFromEnv()

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, secure)
		})
	}
}

func TestLoginMapsBadCredentialsTo401WithoutLeakingWhy(t *testing.T) {
	service, store := serviceWithUser(t, "max@example.com", "correct horse battery")

	for _, tc := range []struct {
		name string
		body string
	}{
		{"falsches Passwort", `{"email":"max@example.com","password":"falsch aber lang"}`},
		{"unbekannte Mail", `{"email":"nobody@example.com","password":"correct horse battery"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveLogin(NewHandler(service, true), tc.body)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.NotEqual(t, http.StatusNotFound, rec.Code)

			assert.JSONEq(t, `{"error":"invalid email or password"}`, rec.Body.String())
			assert.Empty(t, rec.Result().Cookies())
		})
	}

	assert.Empty(t, store.created, "fehlgeschlagener Login darf keine Session anlegen")
}

func TestLoginMapsStoreFailureTo500(t *testing.T) {
	store := &fakeSessionStore{err: fmt.Errorf("datenbank weg")}
	service := NewService(&fakeRepo{}, store)
	_, err := service.Register(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	rec := serveLogin(NewHandler(service, true), `{"email":"max@example.com","password":"correct horse battery"}`)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "datenbank weg", "interne Fehler gehoeren ins Log, nicht in die Antwort")
}

func TestLoginRejectsInvalidBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want int
	}{
		{"kaputtes JSON", `{"email":`, http.StatusBadRequest},
		{"keine Mail", `{"email":"keine-mail","password":"egal"}`, http.StatusUnprocessableEntity},
		{"Passwort fehlt", `{"email":"max@example.com"}`, http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveLogin(NewHandler(nil, true), tc.body)

			assert.Equal(t, tc.want, rec.Code)
		})
	}
}

func TestLoginDoesNotEnforcePasswordPolicy(t *testing.T) {
	service, _ := serviceWithUser(t, "max@example.com", "correct horse battery")

	rec := serveLogin(NewHandler(service, true), `{"email":"max@example.com","password":"kurz"}`)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
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
			h := NewHandler(NewService(&fakeRepo{}, nil), true)

			c, rec := postJSON(fmt.Sprintf(`{"email":"max@example.com","password":%q}`, tc.password))
			h.Register(c)

			assert.Equal(t, tc.want, rec.Code)
		})
	}
}
