package invoice

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService() *Service {
	return NewService(NewRepository())
}

// ptr is a shorthand for taking the address of a literal, which Go does not
// allow directly.
func ptr[T any](v T) *T { return &v }

// draftInvoice returns an unsaved invoice that satisfies every requirement for
// issuing, so tests can vary single fields without repeating the whole struct.
func draftInvoice() Invoice {
	return Invoice{
		Status:       StatusDraft,
		ServiceDate:  time.Now().Add(-24 * time.Hour),
		PaymentDueAt: time.Now().Add(14 * 24 * time.Hour),
		Sender: Issuer{
			Contact: Contact{Name: "Sender GmbH", Street: "Hauptstr. 1", Zip: "70173", City: "Stuttgart", Country: "DE"},
			VatID:   "DE123456789",
			IBAN:    "DE89370400440532013000",
		},
		Recipient: Contact{Name: "Recipient GmbH", Street: "Nebenstr. 2", Zip: "70174", City: "Stuttgart", Country: "DE"},
		Items: []LineItem{
			{Description: "Beratung", Quantity: 2, UnitPrice: 100},
		},
		VATRate: 0.19,
	}
}

func seedDraftInvoice(t *testing.T, s *Service) Invoice {
	created, err := s.Create(draftInvoice())
	require.NoError(t, err)
	return created
}

func TestPartialUpdate_NotesOnly_LeavesOtherFieldsUnchanged(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	newNotes := "Bitte bis Ende des Monats zahlen"
	updated, err := s.PartialUpdate(created.ID, InvoicePatch{Notes: &newNotes})

	require.NoError(t, err)
	assert.Equal(t, newNotes, updated.Notes)
	assert.Equal(t, created.Recipient, updated.Recipient)
	assert.Equal(t, created.Items, updated.Items)
	assert.Equal(t, created.GrossTotal, updated.GrossTotal)
}

func TestPartialUpdate_RecalculatesTotals(t *testing.T) {
	tests := []struct {
		name      string
		items     []LineItem
		vatRate   float64
		wantNet   Money
		wantVAT   Money
		wantGross Money
	}{
		{
			name:      "standard VAT rate",
			items:     []LineItem{{Description: "Beratung", Quantity: 3, UnitPrice: 15000}},
			vatRate:   0.19,
			wantNet:   45000,
			wantVAT:   8550,
			wantGross: 53550,
		},
		{
			name:      "reduced VAT rate",
			items:     []LineItem{{Description: "Buch", Quantity: 1, UnitPrice: 2000}},
			vatRate:   0.07,
			wantNet:   2000,
			wantVAT:   140,
			wantGross: 2140,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService()
			created := seedDraftInvoice(t, s)

			patch := InvoicePatch{Items: &tt.items, VATRate: &tt.vatRate}
			updated, err := s.PartialUpdate(created.ID, patch)

			require.NoError(t, err)
			assert.Equal(t, tt.wantNet, updated.NetTotal)
			assert.Equal(t, tt.wantVAT, updated.VATAmount)
			assert.Equal(t, tt.wantGross, updated.GrossTotal)
		})
	}
}

func TestPartialUpdate_UnknownID_ReturnsNotFound(t *testing.T) {
	s := newTestService()

	notes := "egal"
	_, err := s.PartialUpdate("does-not-exist", InvoicePatch{Notes: &notes})

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCreate_DraftHasNoInvoiceNumber(t *testing.T) {
	s := newTestService()

	created, err := s.Create(draftInvoice())

	require.NoError(t, err)
	assert.Equal(t, StatusDraft, created.Status)
	assert.Nil(t, created.InvoiceNumber, "a draft must not carry a number before it is issued")
}

func TestCreate_IgnoresClientSuppliedInvoiceNumber(t *testing.T) {
	s := newTestService()

	draft := draftInvoice()
	draft.InvoiceNumber = ptr("INV-2026-0001")

	created, err := s.Create(draft)

	require.NoError(t, err)
	assert.Nil(t, created.InvoiceNumber, "the number is server owned and must be discarded on create")
}

func TestUpdate_PreservesServerManagedFields(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	tampered := created
	tampered.Status = StatusPaid
	tampered.InvoiceNumber = ptr("HACKED-001")
	tampered.CreatedAt = time.Time{}

	updated, err := s.Update(created.ID, tampered)

	require.NoError(t, err)
	assert.Equal(t, created.Status, updated.Status)
	assert.Nil(t, updated.InvoiceNumber, "a PUT must not be able to set the number on a draft")
	assert.Equal(t, created.CreatedAt, updated.CreatedAt)
}

func TestUpdate_IssuedNumberSurvivesTampering(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	created := seedDraftInvoice(t, s)

	issued, err := s.Issue(created.ID)
	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)

	// Drop back to draft behind the service's back: Update refuses non-drafts,
	// but the number must be preserved for its own sake, not only because the
	// status check happens to reject the call.
	stored, err := repo.GetByID(issued.ID)
	require.NoError(t, err)
	stored.Status = StatusDraft
	_, err = repo.Update(stored.ID, func(Invoice, func(time.Time) (int, error)) (Invoice, error) {
		return stored, nil
	})
	require.NoError(t, err)

	tampered := stored
	tampered.InvoiceNumber = ptr("HACKED-001")

	updated, err := s.Update(stored.ID, tampered)

	require.NoError(t, err)
	require.NotNil(t, updated.InvoiceNumber)
	assert.Equal(t, *issued.InvoiceNumber, *updated.InvoiceNumber)
}

func seedIssuedInvoice(t *testing.T, repo *Repository) Invoice {
	created, err := repo.Create(Invoice{
		ID:      "issued-1",
		Status:  StatusIssued,
		VATRate: 0.19,
		Items:   []LineItem{{Description: "Beratung", Quantity: 1, UnitPrice: 100}},
	})
	require.NoError(t, err)
	return created
}

func TestUpdate_NonDraft_ReturnsNotUpdatable(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	issued := seedIssuedInvoice(t, repo)

	_, err := s.Update(issued.ID, issued)

	assert.ErrorIs(t, err, ErrNotUpdatable)
}

func TestPartialUpdate_NonDraft_ReturnsNotUpdatable(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	issued := seedIssuedInvoice(t, repo)

	notes := "egal"
	_, err := s.PartialUpdate(issued.ID, InvoicePatch{Notes: &notes})

	assert.ErrorIs(t, err, ErrNotUpdatable)
}

func TestDelete_NonDraft_ReturnsNotDeletable(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	issued := seedIssuedInvoice(t, repo)

	err := s.Delete(issued.ID)

	assert.ErrorIs(t, err, ErrNotDeletable)
}

func TestDelete_Draft_Succeeds(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	err := s.Delete(created.ID)
	require.NoError(t, err)

	_, getErr := s.GetByID(created.ID)
	assert.ErrorIs(t, getErr, ErrNotFound)
}

func TestPartialUpdate_InvalidData_ReturnsInvalidInput(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	items := []LineItem{{Description: "X", Quantity: -1, UnitPrice: 10}}
	_, err := s.PartialUpdate(created.ID, InvoicePatch{Items: &items})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdate_InvalidData_ReturnsInvalidInput(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	replacement := created
	replacement.VATRate = 1.5

	_, err := s.Update(created.ID, replacement)

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdate_FractionalVATRate_ReturnsInvalidInput(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	replacement := created
	replacement.VATRate = 0.195

	_, err := s.Update(created.ID, replacement)

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestIssue_Draft_SetsNumberAndTimestamp(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	issued, err := s.Issue(created.ID)

	require.NoError(t, err)
	assert.Equal(t, StatusIssued, issued.Status)
	assert.False(t, issued.IssuedAt.IsZero())
	require.NotNil(t, issued.InvoiceNumber)
	assert.Regexp(t, `\d{4}-\d{4}$`, *issued.InvoiceNumber)
}

func TestIssue_AssignsSequentialNumbers(t *testing.T) {
	s := newTestService()
	first := seedDraftInvoice(t, s)
	second := seedDraftInvoice(t, s)

	a, err := s.Issue(first.ID)
	require.NoError(t, err)
	b, err := s.Issue(second.ID)
	require.NoError(t, err)

	require.NotNil(t, a.InvoiceNumber)
	require.NotNil(t, b.InvoiceNumber)
	assert.Regexp(t, `\d{4}-0001$`, *a.InvoiceNumber)
	assert.Regexp(t, `\d{4}-0002$`, *b.InvoiceNumber)
}

func TestIssue_AlreadyIssued_ReturnsInvalidTransition(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	_, err := s.Issue(created.ID)
	require.NoError(t, err)

	_, err = s.Issue(created.ID)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestIssue_MissingRequiredFields_ReturnsAllMissingAtOnce(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*Invoice)
		wantMissing []string
	}{
		{
			name: "missing everything required for issue",
			mutate: func(inv *Invoice) {
				inv.ServiceDate = time.Time{}
				inv.Sender.IBAN = ""
				inv.Sender.VatID = ""
				inv.Sender.TaxNumber = ""
			},
			wantMissing: []string{"serviceDate", "senderIban", "senderVatId or senderTaxNumber"},
		},
		{
			name:        "missing only serviceDate",
			mutate:      func(inv *Invoice) { inv.ServiceDate = time.Time{} },
			wantMissing: []string{"serviceDate"},
		},
		{
			name:        "missing only senderIban",
			mutate:      func(inv *Invoice) { inv.Sender.IBAN = "" },
			wantMissing: []string{"senderIban"},
		},
		{
			name:        "vatId present but taxNumber empty is still sufficient",
			mutate:      func(inv *Invoice) { inv.Sender.VatID = "DE123456789"; inv.Sender.TaxNumber = "" },
			wantMissing: nil,
		},
		{
			name:        "taxNumber present but vatId empty is still sufficient",
			mutate:      func(inv *Invoice) { inv.Sender.VatID = ""; inv.Sender.TaxNumber = "12/345/67890" },
			wantMissing: nil,
		},
		{
			name:        "both vatId and taxNumber empty",
			mutate:      func(inv *Invoice) { inv.Sender.VatID = ""; inv.Sender.TaxNumber = "" },
			wantMissing: []string{"senderVatId or senderTaxNumber"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService()
			created := seedDraftInvoice(t, s)

			replacement := created
			tt.mutate(&replacement)
			_, err := s.Update(created.ID, replacement)
			require.NoError(t, err)

			issued, err := s.Issue(created.ID)

			if tt.wantMissing == nil {
				assert.NoError(t, err)
				assert.Equal(t, StatusIssued, issued.Status)
				return
			}

			var mfe *MissingFieldsError
			require.True(t, errors.As(err, &mfe), "expected *MissingFieldsError, got %T: %v", err, err)
			assert.ElementsMatch(t, tt.wantMissing, mfe.Fields)
		})
	}
}

func TestIssue_EmptyCurrency_ReturnsMissingCurrency(t *testing.T) {
	// Bypasses Service.Update's defaulting to cover rows that reach the repo
	// with an empty currency through another path (e.g. a future write path).
	repo := NewRepository()
	s := NewService(repo)
	created, err := repo.Create(Invoice{
		Status:      StatusDraft,
		ServiceDate: time.Now().Add(-24 * time.Hour),
		Currency:    "",
		Sender: Issuer{
			Contact: Contact{Name: "Sender GmbH", Street: "Hauptstr. 1", Zip: "70173", City: "Stuttgart", Country: "DE"},
			VatID:   "DE123456789",
			IBAN:    "DE89370400440532013000",
		},
		Recipient: Contact{Name: "Recipient GmbH", Street: "Nebenstr. 2", Zip: "70174", City: "Stuttgart", Country: "DE"},
		Items:     []LineItem{{Description: "Beratung", Quantity: 1, UnitPrice: 100}},
		VATRate:   0.19,
	})
	require.NoError(t, err)

	_, err = s.Issue(created.ID)

	var mfe *MissingFieldsError
	require.True(t, errors.As(err, &mfe), "expected *MissingFieldsError, got %T: %v", err, err)
	assert.ElementsMatch(t, []string{"currency"}, mfe.Fields)
}

func TestUpdate_OmittedCurrency_DefaultsToEUR(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)
	require.Equal(t, "EUR", created.Currency)

	replacement := created
	replacement.Currency = "" // client PUT that omits currency must not wipe it

	updated, err := s.Update(created.ID, replacement)
	require.NoError(t, err)
	assert.Equal(t, "EUR", updated.Currency)

	issued, err := s.Issue(updated.ID)
	require.NoError(t, err)
	assert.Equal(t, "EUR", issued.Currency)
}

func TestUpdate_OmittedCurrency_PreservesNonEURCurrency(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	created.Currency = "USD"
	created, err := s.Update(created.ID, created)
	require.NoError(t, err)
	require.Equal(t, "USD", created.Currency)

	replacement := created
	replacement.Currency = "" // client PUT that omits currency must not reset it to EUR

	updated, err := s.Update(created.ID, replacement)
	require.NoError(t, err)
	assert.Equal(t, "USD", updated.Currency)
}

func TestIssue_UnknownID_ReturnsNotFound(t *testing.T) {
	s := newTestService()

	_, err := s.Issue("does-not-exist")

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestIssue_ThenUpdate_ReturnsNotUpdatable(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	issued, err := s.Issue(created.ID)
	require.NoError(t, err)

	notes := "zu spät"
	_, err = s.PartialUpdate(issued.ID, InvoicePatch{Notes: &notes})

	assert.ErrorIs(t, err, ErrNotUpdatable)
}

func TestNextCounter_ResetsOnNewYear(t *testing.T) {
	repo := NewRepository()

	first, err := repo.nextCounter(time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, first)

	second, err := repo.nextCounter(time.Date(2025, 12, 31, 23, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 2, second)

	third, err := repo.nextCounter(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, third)
}

func TestIssue_NumberYearFollowsIssuedAt(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	// 23:59:59 UTC on New Year's Eve is already the next year in Europe/Berlin.
	// Number and issuedAt come from one timestamp, so both have to move.
	s.now = func() time.Time {
		return time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	issued, err := s.Issue(created.ID)

	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)
	assert.Equal(t, "INV-2026-0001", *issued.InvoiceNumber)
	assert.Equal(t, 2026, issued.IssuedAt.Year())
}
