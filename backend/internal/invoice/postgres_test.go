package invoice

import (
	"context"
	"testing"

	"github.com/Mirac61/Invoice/backend/internal/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresGetByID(t *testing.T) {
	pool, err := db.NewPool(context.Background(), "postgres://invoice_user:bevvyr-9fexki-qyhQup@localhost:5432/invoice_db?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	repo := NewPostgresRepository(pool)

	invoice := Invoice{
		ID:            uuid.NewString(),
		InvoiceNumber: "INV-TEST-" + uuid.NewString(),
		Status:        StatusDraft,
		Sender:        Issuer{Contact: Contact{Name: "Sender", Street: "S1", Zip: "111", City: "C1", Country: "DE"}},
		Recipient:     Contact{Name: "Recipient", Street: "S2", Zip: "222", City: "C2", Country: "DE"},
		Items: []LineItem{
			{ID: uuid.NewString(), Position: 1, Description: "Test Item", Quantity: 1, UnitPrice: 50, Total: 50},
		},
		VATRate: 0.19,
	}

	created, err := repo.Create(invoice)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(created.ID) })

	fetched, err := repo.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.InvoiceNumber, fetched.InvoiceNumber)
	assert.Len(t, fetched.Items, 1)
	assert.Equal(t, "Test Item", fetched.Items[0].Description)
}
