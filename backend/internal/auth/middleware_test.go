package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authRouter(t *testing.T, ttl time.Duration) (*gin.Engine, *fakeSessionStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store := &fakeSessionStore{}
	service := NewServiceWithSessionTTL(&fakeRepo{}, store, ttl)
	_, err := service.Register(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)
	h := NewHandler(service, true)

	r := gin.New()
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/logout", h.Logout)
	r.GET("/api/auth/me", RequireAuth(store, ttl, true), h.Me)
	return r, store
}

func doLogin(t *testing.T, r *gin.Engine) *http.Cookie {
	t.Helper()
	body := `{"email":"max@example.com","password":"correct horse battery"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
	return sessionCookie(t, rec)
}

func do(r *gin.Engine, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestMeReturnsCurrentUser(t *testing.T) {
	r, _ := authRouter(t, defaultSessionTTL)

	rec := do(r, http.MethodGet, "/api/auth/me", doLogin(t, r))

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "max@example.com", body["email"])
	assert.NotEmpty(t, body["id"])
	assert.NotContains(t, rec.Body.String(), "argon2", "der Hash gehoert nicht in die Antwort")
}

func TestMeRejectsMissingOrManipulatedCookie(t *testing.T) {
	r, _ := authRouter(t, defaultSessionTTL)
	valid := doLogin(t, r)

	for _, tc := range []struct {
		name   string
		cookie *http.Cookie
	}{
		{"kein Cookie", nil},
		{"leeres Cookie", &http.Cookie{Name: "session", Value: ""}},
		{"fremdes Token", &http.Cookie{Name: "session", Value: "irgendwas"}},
		{"veraendertes Token", &http.Cookie{Name: "session", Value: "A" + valid.Value[1:]}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(r, http.MethodGet, "/api/auth/me", tc.cookie)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestLogoutRemovesSessionAndCookie(t *testing.T) {
	r, store := authRouter(t, defaultSessionTTL)
	cookie := doLogin(t, r)
	require.Len(t, store.created, 1)

	rec := do(r, http.MethodPost, "/api/auth/logout", cookie)

	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, store.created, "die Session-Zeile muss weg sein")

	cleared := sessionCookie(t, rec)
	assert.Empty(t, cleared.Value)
	assert.Negative(t, cleared.MaxAge, "Max-Age=0 loescht, 0 wuerde nur das Attribut weglassen")
	assert.Equal(t, "/", cleared.Path)
	assert.True(t, cleared.Secure)
	assert.True(t, cleared.HttpOnly)

	assert.Equal(t, http.StatusUnauthorized, do(r, http.MethodGet, "/api/auth/me", cookie).Code)
}

func TestLogoutWithoutSessionStillAnswers204(t *testing.T) {
	r, _ := authRouter(t, defaultSessionTTL)

	for _, tc := range []struct {
		name   string
		cookie *http.Cookie
	}{
		{"kein Cookie", nil},
		{"unbekanntes Token", &http.Cookie{Name: "session", Value: "irgendwas"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(r, http.MethodPost, "/api/auth/logout", tc.cookie)

			require.Equal(t, http.StatusNoContent, rec.Code)
			assert.Negative(t, sessionCookie(t, rec).MaxAge)
		})
	}
}

func TestExpiredSessionIsRejectedAndDeleted(t *testing.T) {
	r, store := authRouter(t, defaultSessionTTL)
	cookie := doLogin(t, r)

	require.Len(t, store.created, 1)
	store.created[0].ExpiresAt = time.Now().UTC().Add(-time.Second)

	rec := do(r, http.MethodGet, "/api/auth/me", cookie)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, store.created, "abgelaufene Sessions raeumt der Zugriff selbst weg")
}

func TestSessionsOfTheSameUserAreIndependent(t *testing.T) {
	r, store := authRouter(t, defaultSessionTTL)
	first := doLogin(t, r)
	second := doLogin(t, r)

	require.Len(t, store.created, 2)
	require.NotEqual(t, first.Value, second.Value)

	require.Equal(t, http.StatusNoContent, do(r, http.MethodPost, "/api/auth/logout", first).Code)

	assert.Equal(t, http.StatusUnauthorized, do(r, http.MethodGet, "/api/auth/me", first).Code)
	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/auth/me", second).Code,
		"das zweite Geraet darf vom Logout nichts merken")
}

func TestSlidingRenewalIsThrottled(t *testing.T) {
	const ttl = time.Hour
	r, store := authRouter(t, ttl)
	cookie := doLogin(t, r)
	require.Len(t, store.created, 1)

	store.created[0].ExpiresAt = time.Now().UTC().Add(ttl - 20*time.Minute)

	require.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/auth/me", cookie).Code)

	require.Equal(t, 1, store.touched)
	renewed := store.created[0].ExpiresAt
	assert.WithinDuration(t, time.Now().UTC().Add(ttl), renewed, time.Minute)

	require.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/auth/me", cookie).Code)

	assert.Equal(t, 1, store.touched, "der zweite Request liegt innerhalb der 15 Minuten")
	assert.Equal(t, renewed, store.created[0].ExpiresAt)
}

func TestRenewalKeepsSessionUsable(t *testing.T) {
	const ttl = time.Hour
	r, store := authRouter(t, ttl)
	cookie := doLogin(t, r)

	store.created[0].ExpiresAt = time.Now().UTC().Add(time.Minute)

	require.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/auth/me", cookie).Code)
	assert.WithinDuration(t, time.Now().UTC().Add(ttl), store.created[0].ExpiresAt, time.Minute)
}

func TestRenewalRefreshesTheCookie(t *testing.T) {
	const ttl = time.Hour
	r, store := authRouter(t, ttl)
	cookie := doLogin(t, r)

	store.created[0].ExpiresAt = time.Now().UTC().Add(ttl - 20*time.Minute)

	rec := do(r, http.MethodGet, "/api/auth/me", cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, store.touched)

	refreshed := sessionCookie(t, rec)
	assert.Equal(t, cookie.Value, refreshed.Value)
	assert.Equal(t, int(ttl.Seconds()), refreshed.MaxAge)
	assert.True(t, refreshed.HttpOnly)
	assert.True(t, refreshed.Secure)
}

func TestRequestWithoutRenewalKeepsTheCookieUntouched(t *testing.T) {
	r, store := authRouter(t, time.Hour)
	cookie := doLogin(t, r)

	rec := do(r, http.MethodGet, "/api/auth/me", cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 0, store.touched)

	assert.Empty(t, rec.Result().Cookies())
}

func TestRenewalWorksForTTLBelowMaxRenewInterval(t *testing.T) {
	const ttl = 10 * time.Minute
	r, store := authRouter(t, ttl)
	cookie := doLogin(t, r)

	store.created[0].ExpiresAt = time.Now().UTC().Add(ttl - 6*time.Minute)

	require.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/auth/me", cookie).Code)
	assert.Equal(t, 1, store.touched)
}

func TestLogoutReportsFailedRevocation(t *testing.T) {
	r, store := authRouter(t, time.Hour)
	cookie := doLogin(t, r)
	store.err = errors.New("boom")

	rec := do(r, http.MethodPost, "/api/auth/logout", cookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, -1, sessionCookie(t, rec).MaxAge, "das Cookie beim Aufrufer muss trotzdem weg")
}

func TestMeIsUnreachableWithoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSessionStore{}
	h := NewHandler(NewService(&fakeRepo{}, store), true)

	r := gin.New()
	r.GET("/api/auth/me", h.Me)

	rec := do(r, http.MethodGet, "/api/auth/me", nil)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
