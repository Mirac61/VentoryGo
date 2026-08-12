package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// tenantSessionStore kennt zwei feste Sessions, damit sich zwei Nutzer im
// selben Router unterscheiden lassen. Der Store bekommt von RequireAuth nur
// den Hash zu sehen; die Zuordnung laeuft deshalb ueber denselben sha256, den
// auth.hashToken bildet.
type tenantSessionStore struct {
	users map[string]uuid.UUID
}

func tokenKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s tenantSessionStore) Get(_ context.Context, hash []byte) (auth.Session, error) {
	userID, ok := s.users[hex.EncodeToString(hash)]
	if !ok {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	return auth.Session{
		TokenHash: hash,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

func (tenantSessionStore) Create(context.Context, auth.Session) error           { return nil }
func (tenantSessionStore) Touch(context.Context, []byte, time.Time) error       { return nil }
func (tenantSessionStore) Delete(context.Context, []byte) error                 { return nil }
func (tenantSessionStore) DeleteExpiredByUser(context.Context, uuid.UUID) error { return nil }

const (
	tokenA = "token-a"
	tokenB = "token-b"
)

func tenantRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	sessions := tenantSessionStore{users: map[string]uuid.UUID{
		tokenKey(tokenA): uuid.New(),
		tokenKey(tokenB): uuid.New(),
	}}
	invoices := invoice.NewHandler(invoice.NewService(invoice.NewRepository()))
	hasher, err := auth.NewHasher(1)
	require.NoError(t, err)
	authService := auth.NewServiceWithSessionTTL(stubUserRepo{}, sessions, time.Hour, hasher)
	authHandler := auth.NewHandler(authService, true)

	return newRouter(invoices, authHandler, sessions, time.Hour, true)
}

func asUser(r *gin.Engine, token, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

const invoiceBody = `{
	"paymentDueAt": "2030-01-01T00:00:00Z",
	"serviceDate": "2029-12-01T00:00:00Z",
	"sender": {"name": "Sender GmbH", "street": "Hauptstr. 1", "zip": "70173", "city": "Stuttgart", "country": "DE",
	           "vatId": "DE123456789", "iban": "DE89370400440532013000"},
	"recipient": {"name": "Recipient GmbH", "street": "Nebenstr. 2", "zip": "70174", "city": "Stuttgart", "country": "DE"},
	"items": [{"description": "Beratung", "quantity": 2, "unitPrice": 100}],
	"vatRate": 0.19
}`

func createAs(t *testing.T, r *gin.Engine, token string) string {
	t.Helper()

	rec := asUser(r, token, http.MethodPost, "/api/invoices", invoiceBody)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.NotEmpty(t, created.ID)
	return created.ID
}

// Eine fremde Rechnung muss sich wie eine nicht existierende verhalten: 404 auf
// jedem Weg, und danach steht sie beim Eigentuemer unveraendert da.
func TestForeignInvoiceIsNotReachableOverHTTP(t *testing.T) {
	r := tenantRouter(t)
	id := createAs(t, r, tokenA)

	foreign := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/invoices/" + id, ""},
		{http.MethodPut, "/api/invoices/" + id, invoiceBody},
		{http.MethodPatch, "/api/invoices/" + id, `{"notes":"fremd"}`},
		{http.MethodDelete, "/api/invoices/" + id, ""},
		{http.MethodPost, "/api/invoices/" + id + "/issue", ""},
	}

	for _, route := range foreign {
		t.Run(route.method, func(t *testing.T) {
			rec := asUser(r, tokenB, route.method, route.path, route.body)
			assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		})
	}

	rec := asUser(r, tokenA, http.MethodGet, "/api/invoices/"+id, "")
	require.Equal(t, http.StatusOK, rec.Code)

	var after struct {
		Status        string  `json:"status"`
		Notes         string  `json:"notes"`
		InvoiceNumber *string `json:"invoiceNumber"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &after))
	assert.Equal(t, "draft", after.Status, "der Fremdzugriff darf den Status nicht veraendert haben")
	assert.Empty(t, after.Notes, "PATCH von B darf die Notizen nicht gesetzt haben")
	assert.Nil(t, after.InvoiceNumber, "POST /issue von B darf keine Nummer gezogen haben")
}

func TestListContainsOnlyOwnInvoices(t *testing.T) {
	r := tenantRouter(t)
	idA := createAs(t, r, tokenA)
	idB := createAs(t, r, tokenB)

	ids := func(token string) []string {
		rec := asUser(r, token, http.MethodGet, "/api/invoices", "")
		require.Equal(t, http.StatusOK, rec.Code)

		var invoices []struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &invoices))

		out := make([]string, 0, len(invoices))
		for _, invoice := range invoices {
			out = append(out, invoice.ID)
		}
		return out
	}

	assert.Equal(t, []string{idA}, ids(tokenA))
	assert.Equal(t, []string{idB}, ids(tokenB))
}

// owner_id darf nicht per Request setzbar sein: das Feld traegt json:"-", ein
// mitgeschicktes ownerId wird also ignoriert.
func TestOwnerFromBodyIsIgnored(t *testing.T) {
	r := tenantRouter(t)

	body := strings.TrimSuffix(strings.TrimSpace(invoiceBody), "}") +
		`, "ownerId": "` + uuid.NewString() + `", "OwnerID": "` + uuid.NewString() + `"}`

	rec := asUser(r, tokenA, http.MethodPost, "/api/invoices", body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	// Die Rechnung gehoert A, nicht dem mitgeschickten Fremd-Owner: B sieht sie
	// nicht, A schon.
	assert.Equal(t, http.StatusNotFound,
		asUser(r, tokenB, http.MethodGet, "/api/invoices/"+created.ID, "").Code)
	assert.Equal(t, http.StatusOK,
		asUser(r, tokenA, http.MethodGet, "/api/invoices/"+created.ID, "").Code)

	// Kleingeschrieben verglichen, damit der Test nicht nur an einer Schreibweise
	// haengt: faellt json:"-" weg, heisst das Feld "OwnerID", ein Tag koennte es
	// "ownerId" oder "owner_id" nennen.
	assert.NotContains(t, strings.ToLower(rec.Body.String()), "owner",
		"der Owner darf in keiner Antwort auftauchen")
}
