package invoice

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() (*gin.Engine, *Service) {
	gin.SetMode(gin.TestMode)
	repo := NewRepository()
	service := NewService(repo)
	handler := NewHandler(service)

	r := gin.New()
	// Steht hier fuer RequireAuth: die Handler lesen den Owner aus dem Context,
	// die Session selbst ist nicht Gegenstand dieser Tests.
	r.Use(func(c *gin.Context) { c.Set("user_id", testOwnerID) })
	r.POST("/api/invoices", handler.Create)
	r.POST("/api/invoices/:id/issue", handler.Issue)
	r.GET("/api/invoices", handler.GetAll)
	r.GET("/api/invoices/:id", handler.GetByID)
	r.DELETE("/api/invoices/:id", handler.Delete)
	r.PUT("/api/invoices/:id", handler.Update)
	r.PATCH("/api/invoices/:id", handler.PartialUpdate)
	return r, service
}

func validInvoiceBody() map[string]any {
	return map[string]any{
		"paymentDueAt": time.Now().Add(14 * 24 * time.Hour),
		"serviceDate":  time.Now().Add(-24 * time.Hour),
		"sender": map[string]any{
			"name": "Sender GmbH", "street": "Hauptstr. 1", "zip": "70173", "city": "Stuttgart", "country": "DE",
			"vatId": "DE123456789", "iban": "DE89370400440532013000",
		},
		"recipient": map[string]any{"name": "Recipient GmbH", "street": "Nebenstr. 2", "zip": "70174", "city": "Stuttgart", "country": "DE"},
		"items": []map[string]any{{"description": "Beratung", "quantity": 2, "unitPrice": 100, "vatRate": 1900}},
	}
}

func doRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// helper: create an invoice through the API and return its ID
func createInvoice(t *testing.T, r *gin.Engine) string {
	w := doRequest(r, http.MethodPost, "/api/invoices", validInvoiceBody())
	require.Equal(t, http.StatusCreated, w.Code)
	var created Invoice
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	return created.ID
}

func TestCreate(t *testing.T) {
	t.Run("valid returns 201 and forces draft", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		body["status"] = "paid" // client tries to force status

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		require.Equal(t, http.StatusCreated, w.Code)
		var created Invoice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
		assert.Equal(t, StatusDraft, created.Status)
		assert.NotEmpty(t, created.ID)
	})

	t.Run("draft serializes invoiceNumber as null", func(t *testing.T) {
		r, _ := setupRouter()

		w := doRequest(r, http.MethodPost, "/api/invoices", validInvoiceBody())

		require.Equal(t, http.StatusCreated, w.Code)
		var raw map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
		value, present := raw["invoiceNumber"]
		require.True(t, present, "clients rely on the key being there")
		assert.Nil(t, value, "a draft must serialize as null, not as an empty string")
	})

	t.Run("client supplied invoiceNumber is discarded", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		body["invoiceNumber"] = "INV-2026-0001" // client tries to pick its own number

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		require.Equal(t, http.StatusCreated, w.Code)
		var raw map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
		assert.Nil(t, raw["invoiceNumber"])
	})

	t.Run("missing required field returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		delete(body, "recipient")

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative quantity returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		body["items"] = []map[string]any{{"description": "X", "quantity": -1, "unitPrice": 100, "vatRate": 1900}}

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty items returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		body["items"] = []map[string]any{}

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("malformed sender IBAN returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		sender := body["sender"].(map[string]any)
		sender["iban"] = "NOT-A-VALID-IBAN"

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("draft with empty sender IBAN is still allowed", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		sender := body["sender"].(map[string]any)
		delete(sender, "iban")

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("unknown currency code returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		body["currency"] = "XYZ"

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("draft without currency defaults to EUR", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()

		w := doRequest(r, http.MethodPost, "/api/invoices", body)

		require.Equal(t, http.StatusCreated, w.Code)
		var created Invoice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
		assert.Equal(t, "EUR", created.Currency)
	})
}

func TestGetByID(t *testing.T) {
	t.Run("existing returns 200", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)

		w := doRequest(r, http.MethodGet, "/api/invoices/"+id, nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("unknown returns 404", func(t *testing.T) {
		r, _ := setupRouter()

		w := doRequest(r, http.MethodGet, "/api/invoices/does-not-exist", nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestGetAll(t *testing.T) {
	t.Run("empty returns 200 and empty array", func(t *testing.T) {
		r, _ := setupRouter()

		w := doRequest(r, http.MethodGet, "/api/invoices", nil)

		require.Equal(t, http.StatusOK, w.Code)
		var invoices []Invoice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invoices))
		assert.Empty(t, invoices)
	})

	t.Run("returns all created invoices", func(t *testing.T) {
		r, _ := setupRouter()
		createInvoice(t, r)
		createInvoice(t, r)

		w := doRequest(r, http.MethodGet, "/api/invoices", nil)

		require.Equal(t, http.StatusOK, w.Code)
		var invoices []Invoice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invoices))
		assert.Len(t, invoices, 2)
	})
}

func TestDelete(t *testing.T) {
	t.Run("draft returns 204", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)

		w := doRequest(r, http.MethodDelete, "/api/invoices/"+id, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("unknown returns 404", func(t *testing.T) {
		r, _ := setupRouter()

		w := doRequest(r, http.MethodDelete, "/api/invoices/does-not-exist", nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateHandler(t *testing.T) {
	t.Run("valid returns 200", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)

		w := doRequest(r, http.MethodPut, "/api/invoices/"+id, validInvoiceBody())

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("cannot set invoiceNumber on a draft", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)
		body := validInvoiceBody()
		body["invoiceNumber"] = "INV-2026-0001"

		w := doRequest(r, http.MethodPut, "/api/invoices/"+id, body)

		require.Equal(t, http.StatusOK, w.Code)
		var raw map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
		assert.Nil(t, raw["invoiceNumber"])
	})

	t.Run("unknown returns 404", func(t *testing.T) {
		r, _ := setupRouter()

		w := doRequest(r, http.MethodPut, "/api/invoices/does-not-exist", validInvoiceBody())

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)
		body := validInvoiceBody()
		delete(body, "sender")

		w := doRequest(r, http.MethodPut, "/api/invoices/"+id, body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPartialUpdateHandler(t *testing.T) {
	t.Run("valid notes patch returns 200", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)

		w := doRequest(r, http.MethodPatch, "/api/invoices/"+id, map[string]any{"notes": "updated"})

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("unknown returns 404", func(t *testing.T) {
		r, _ := setupRouter()

		w := doRequest(r, http.MethodPatch, "/api/invoices/does-not-exist", map[string]any{"notes": "x"})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("negative quantity in patched items returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)
		body := map[string]any{"items": []map[string]any{{"description": "X", "quantity": -5, "unitPrice": 10, "vatRate": 1900}}}

		w := doRequest(r, http.MethodPatch, "/api/invoices/"+id, body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("zero paymentDueAt returns 400", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)

		w := doRequest(r, http.MethodPatch, "/api/invoices/"+id, map[string]any{
			"paymentDueAt": "0001-01-01T00:00:00Z",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestIssueHandler(t *testing.T) {
	t.Run("draft returns 200 with number", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)

		w := doRequest(r, http.MethodPost, "/api/invoices/"+id+"/issue", nil)

		require.Equal(t, http.StatusOK, w.Code)
		var issued Invoice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &issued))
		assert.Equal(t, StatusIssued, issued.Status)
		require.NotNil(t, issued.InvoiceNumber)
		assert.NotEmpty(t, *issued.InvoiceNumber)
	})

	t.Run("issued serializes invoiceNumber as a string", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)

		w := doRequest(r, http.MethodPost, "/api/invoices/"+id+"/issue", nil)

		require.Equal(t, http.StatusOK, w.Code)
		var raw map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
		number, ok := raw["invoiceNumber"].(string)
		require.Truef(t, ok, "expected a JSON string, got %#v", raw["invoiceNumber"])
		assert.NotEmpty(t, number)
	})

	t.Run("already issued returns 409", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)
		require.Equal(t, http.StatusOK, doRequest(r, http.MethodPost, "/api/invoices/"+id+"/issue", nil).Code)

		w := doRequest(r, http.MethodPost, "/api/invoices/"+id+"/issue", nil)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("unknown returns 404", func(t *testing.T) {
		r, _ := setupRouter()

		w := doRequest(r, http.MethodPost, "/api/invoices/does-not-exist/issue", nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("patch after issue returns 409", func(t *testing.T) {
		r, _ := setupRouter()
		id := createInvoice(t, r)
		require.Equal(t, http.StatusOK, doRequest(r, http.MethodPost, "/api/invoices/"+id+"/issue", nil).Code)

		w := doRequest(r, http.MethodPatch, "/api/invoices/"+id, map[string]any{"notes": "x"})

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("draft missing §14 UStG fields returns 422 with full field list", func(t *testing.T) {
		r, _ := setupRouter()
		body := validInvoiceBody()
		delete(body, "serviceDate")
		sender := body["sender"].(map[string]any)
		delete(sender, "iban")
		delete(sender, "vatId")

		w := doRequest(r, http.MethodPost, "/api/invoices", body)
		require.Equal(t, http.StatusCreated, w.Code)
		var created Invoice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

		w = doRequest(r, http.MethodPost, "/api/invoices/"+created.ID+"/issue", nil)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var resp struct {
			MissingFields []string `json:"missingFields"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.ElementsMatch(t, []string{"serviceDate", "senderIban", "senderVatId or senderTaxNumber"}, resp.MissingFields)
	})
}
