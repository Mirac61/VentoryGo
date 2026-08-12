package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mirac61/VentoryGo/backend/internal/auth"
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dummyID = "00000000-0000-0000-0000-000000000000"

// Die Attrappen werden nie wirklich befragt: ohne Cookie bricht RequireAuth ab,
// bevor es an den Store geht, und bei kaputtem JSON antwortet der Auth-Handler,
// bevor er das Repository anfasst. Sie existieren nur, damit der Router ohne
// Datenbank gebaut werden kann.
type stubUserRepo struct{}

func (stubUserRepo) Create(context.Context, auth.User) error { return nil }

func (stubUserRepo) FindByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, auth.ErrUserNotFound
}

func (stubUserRepo) FindByID(context.Context, uuid.UUID) (auth.User, error) {
	return auth.User{}, auth.ErrUserNotFound
}

type stubSessionStore struct{}

func (stubSessionStore) Create(context.Context, auth.Session) error { return nil }

func (stubSessionStore) Get(context.Context, []byte) (auth.Session, error) {
	return auth.Session{}, auth.ErrSessionNotFound
}

func (stubSessionStore) Touch(context.Context, []byte, time.Time) error { return nil }

func (stubSessionStore) Delete(context.Context, []byte) error { return nil }

func (stubSessionStore) DeleteExpiredByUser(context.Context, uuid.UUID) error { return nil }

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	sessions := stubSessionStore{}
	invoices := invoice.NewHandler(invoice.NewService(invoice.NewRepository()))
	hasher, err := auth.NewHasher(1)
	require.NoError(t, err)
	authService := auth.NewServiceWithSessionTTL(stubUserRepo{}, sessions, time.Hour, hasher)
	authHandler := auth.NewHandler(authService, true)

	return newRouter(invoices, authHandler, sessions, time.Hour, true)
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

var invoiceRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/invoices"},
	{http.MethodPost, "/api/invoices"},
	{http.MethodGet, "/api/invoices/" + dummyID},
	{http.MethodPut, "/api/invoices/" + dummyID},
	{http.MethodPatch, "/api/invoices/" + dummyID},
	{http.MethodDelete, "/api/invoices/" + dummyID},
	{http.MethodPost, "/api/invoices/" + dummyID + "/issue"},
}

func TestInvoiceEndpointsRequireSession(t *testing.T) {
	r := testRouter(t)

	for _, route := range invoiceRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			rec := do(r, route.method, route.path, "")
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

// Register und Login duerfen nicht in der geschuetzten Gruppe landen. Kaputtes
// JSON statt gueltiger Daten, damit der Test keinen Nutzer anlegt: 400 beweist,
// dass der Handler erreicht wurde, 401 waere die Middleware gewesen.
func TestAuthEndpointsStayPublic(t *testing.T) {
	r := testRouter(t)

	for _, path := range []string{"/api/auth/register", "/api/auth/login"} {
		t.Run(path, func(t *testing.T) {
			rec := do(r, http.MethodPost, path, "{")
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

// fillParams ersetzt Platzhalter wie :id, sonst matcht gin die Route nicht.
func fillParams(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*") {
			segments[i] = dummyID
		}
	}
	return strings.Join(segments, "/")
}

// unprotectedRoutes geht die tatsaechlich registrierten Routen durch und liefert
// alle /api/-Routen ausser /api/auth/*, die ohne Session nicht mit 401 antworten.
// Bewusst als Rueckgabewert statt t.Fatal: nur so laesst sich in
// TestSweepDetectsUnprotectedRoute pruefen, dass der Sweep selbst anschlaegt.
func unprotectedRoutes(r *gin.Engine) []string {
	var open []string
	for _, route := range r.Routes() {
		if !strings.HasPrefix(route.Path, "/api/") || strings.HasPrefix(route.Path, "/api/auth/") {
			continue
		}
		if rec := do(r, route.Method, fillParams(route.Path), ""); rec.Code != http.StatusUnauthorized {
			open = append(open, route.Method+" "+route.Path)
		}
	}
	return open
}

// Faengt Routen ab, die spaeter versehentlich ausserhalb der Gruppe registriert
// werden -- die Tests oben kennen nur die heute bekannten Pfade.
func TestNoAPIRouteIsUnprotected(t *testing.T) {
	assert.Empty(t, unprotectedRoutes(testRouter(t)))
}

func TestSweepDetectsUnprotectedRoute(t *testing.T) {
	r := testRouter(t)
	r.GET("/api/reports", func(c *gin.Context) { c.Status(http.StatusOK) })

	assert.Equal(t, []string{"GET /api/reports"}, unprotectedRoutes(r))
}
