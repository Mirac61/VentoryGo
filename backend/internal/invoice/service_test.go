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

// draftInvoice satisfies every requirement for issuing, so tests can vary
// single fields without repeating the whole struct.
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
			{Description: "Beratung", Quantity: 2, UnitPrice: 100, VatRate: 1900},
		},
	}
}

func seedDraftInvoice(t *testing.T, s *Service) Invoice {
	created, err := s.Create(draftInvoice(), testOwner)
	require.NoError(t, err)
	return created
}

func seedIssuedInvoice(t *testing.T, repo *Repository) Invoice {
	created, err := repo.Create(Invoice{
		ID:     "issued-1",
		Status: StatusIssued,
		Items:  []LineItem{{Description: "Beratung", Quantity: 1, UnitPrice: 100, VatRate: 1900}},
	}, testOwner)
	require.NoError(t, err)
	return created
}

func TestPartialUpdate_NotesOnly_LeavesOtherFieldsUnchanged(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	newNotes := "Bitte bis Ende des Monats zahlen"
	updated, err := s.PartialUpdate(created.ID, InvoicePatch{Notes: &newNotes}, testOwner)

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
		wantNet   Money
		wantVAT   Money
		wantGross Money
	}{
		{
			name:      "standard VAT rate",
			items:     []LineItem{{Description: "Beratung", Quantity: 3, UnitPrice: 15000, VatRate: 1900}},
			wantNet:   45000,
			wantVAT:   8550,
			wantGross: 53550,
		},
		{
			name:      "reduced VAT rate",
			items:     []LineItem{{Description: "Buch", Quantity: 1, UnitPrice: 2000, VatRate: 700}},
			wantNet:   2000,
			wantVAT:   140,
			wantGross: 2140,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newTestService()
			created := seedDraftInvoice(t, s)

			patch := InvoicePatch{Items: &test.items}
			updated, err := s.PartialUpdate(created.ID, patch, testOwner)

			require.NoError(t, err)
			assert.Equal(t, test.wantNet, updated.NetTotal)
			assert.Equal(t, test.wantVAT, updated.VATAmount)
			assert.Equal(t, test.wantGross, updated.GrossTotal)
		})
	}
}

func TestPartialUpdate_UnknownID_ReturnsNotFound(t *testing.T) {
	s := newTestService()

	notes := "egal"
	_, err := s.PartialUpdate("does-not-exist", InvoicePatch{Notes: &notes}, testOwner)

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCreate_DraftHasNoInvoiceNumber(t *testing.T) {
	s := newTestService()

	created, err := s.Create(draftInvoice(), testOwner)

	require.NoError(t, err)
	assert.Equal(t, StatusDraft, created.Status)
	assert.Nil(t, created.InvoiceNumber, "a draft must not carry a number before it is issued")
}

func TestCreate_IgnoresClientSuppliedInvoiceNumber(t *testing.T) {
	s := newTestService()

	draft := draftInvoice()
	draft.InvoiceNumber = new("INV-2026-0001")

	created, err := s.Create(draft, testOwner)

	require.NoError(t, err)
	assert.Nil(t, created.InvoiceNumber, "the number is server owned and must be discarded on create")
}

func TestUpdate_PreservesServerManagedFields(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	tampered := created
	tampered.Status = StatusPaid
	tampered.InvoiceNumber = new("HACKED-001")
	tampered.CreatedAt = time.Time{}

	updated, err := s.Update(created.ID, tampered, testOwner)

	require.NoError(t, err)
	assert.Equal(t, created.Status, updated.Status)
	assert.Nil(t, updated.InvoiceNumber, "a PUT must not be able to set the number on a draft")
	assert.Equal(t, created.CreatedAt, updated.CreatedAt)
}

func TestUpdate_IssuedNumberSurvivesTampering(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	created := seedDraftInvoice(t, s)

	issued, err := s.Issue(created.ID, testOwner)
	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)

	// Drop back to draft behind the service's back, so the number is preserved
	// for its own sake and not just because the status check rejects the call.
	stored, err := repo.GetByID(issued.ID, testOwner)
	require.NoError(t, err)
	stored.Status = StatusDraft
	_, err = repo.Update(stored.ID, func(Invoice, Numbering, func(time.Time) (int, error)) (Invoice, error) {
		return stored, nil
	}, testOwner)
	require.NoError(t, err)

	tampered := stored
	tampered.InvoiceNumber = new("HACKED-001")

	updated, err := s.Update(stored.ID, tampered, testOwner)

	require.NoError(t, err)
	require.NotNil(t, updated.InvoiceNumber)
	assert.Equal(t, *issued.InvoiceNumber, *updated.InvoiceNumber)
}

func TestUpdate_NonDraft_ReturnsNotUpdatable(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	issued := seedIssuedInvoice(t, repo)

	_, err := s.Update(issued.ID, issued, testOwner)

	assert.ErrorIs(t, err, ErrNotUpdatable)
}

func TestPartialUpdate_NonDraft_ReturnsNotUpdatable(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	issued := seedIssuedInvoice(t, repo)

	notes := "egal"
	_, err := s.PartialUpdate(issued.ID, InvoicePatch{Notes: &notes}, testOwner)

	assert.ErrorIs(t, err, ErrNotUpdatable)
}

func TestDelete_NonDraft_ReturnsNotDeletable(t *testing.T) {
	repo := NewRepository()
	s := NewService(repo)
	issued := seedIssuedInvoice(t, repo)

	err := s.Delete(issued.ID, testOwner)

	assert.ErrorIs(t, err, ErrNotDeletable)
}

func TestDelete_Draft_Succeeds(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	err := s.Delete(created.ID, testOwner)
	require.NoError(t, err)

	_, getErr := s.GetByID(created.ID, testOwner)
	assert.ErrorIs(t, getErr, ErrNotFound)
}

func TestPartialUpdate_InvalidData_ReturnsInvalidInput(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	items := []LineItem{{Description: "X", Quantity: -1, UnitPrice: 10}}
	_, err := s.PartialUpdate(created.ID, InvoicePatch{Items: &items}, testOwner)

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdate_InvalidData_ReturnsInvalidInput(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	replacement := created
	replacement.Items[0].VatRate = -1

	_, err := s.Update(created.ID, replacement, testOwner)

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdate_InvalidVatRate_ReturnsInvalidInput(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	replacement := created
	replacement.Items[0].VatRate = -1

	_, err := s.Update(created.ID, replacement, testOwner)

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestIssue_Draft_SetsNumberAndTimestamp(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	issued, err := s.Issue(created.ID, testOwner)

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

	a, err := s.Issue(first.ID, testOwner)
	require.NoError(t, err)
	b, err := s.Issue(second.ID, testOwner)
	require.NoError(t, err)

	require.NotNil(t, a.InvoiceNumber)
	require.NotNil(t, b.InvoiceNumber)
	assert.Regexp(t, `\d{4}-0001$`, *a.InvoiceNumber)
	assert.Regexp(t, `\d{4}-0002$`, *b.InvoiceNumber)
}

func TestIssue_AlreadyIssued_ReturnsInvalidTransition(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	_, err := s.Issue(created.ID, testOwner)
	require.NoError(t, err)

	_, err = s.Issue(created.ID, testOwner)
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newTestService()
			created := seedDraftInvoice(t, s)

			replacement := created
			test.mutate(&replacement)
			_, err := s.Update(created.ID, replacement, testOwner)
			require.NoError(t, err)

			issued, err := s.Issue(created.ID, testOwner)

			if test.wantMissing == nil {
				assert.NoError(t, err)
				assert.Equal(t, StatusIssued, issued.Status)
				return
			}

			var mfe *MissingFieldsError
			require.True(t, errors.As(err, &mfe), "expected *MissingFieldsError, got %T: %v", err, err)
			assert.ElementsMatch(t, test.wantMissing, mfe.Fields)
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
		Items:     []LineItem{{Description: "Beratung", Quantity: 1, UnitPrice: 100, VatRate: 1900}},
	}, testOwner)
	require.NoError(t, err)

	_, err = s.Issue(created.ID, testOwner)

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

	updated, err := s.Update(created.ID, replacement, testOwner)
	require.NoError(t, err)
	assert.Equal(t, "EUR", updated.Currency)

	issued, err := s.Issue(updated.ID, testOwner)
	require.NoError(t, err)
	assert.Equal(t, "EUR", issued.Currency)
}

func TestUpdate_OmittedCurrency_PreservesNonEURCurrency(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	created.Currency = "USD"
	created, err := s.Update(created.ID, created, testOwner)
	require.NoError(t, err)
	require.Equal(t, "USD", created.Currency)

	replacement := created
	replacement.Currency = "" // client PUT that omits currency must not reset it to EUR

	updated, err := s.Update(created.ID, replacement, testOwner)
	require.NoError(t, err)
	assert.Equal(t, "USD", updated.Currency)
}

func TestIssue_UnknownID_ReturnsNotFound(t *testing.T) {
	s := newTestService()

	_, err := s.Issue("does-not-exist", testOwner)

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestIssue_ThenUpdate_ReturnsNotUpdatable(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	issued, err := s.Issue(created.ID, testOwner)
	require.NoError(t, err)

	notes := "zu spät"
	_, err = s.PartialUpdate(issued.ID, InvoicePatch{Notes: &notes}, testOwner)

	assert.ErrorIs(t, err, ErrNotUpdatable)
}

func TestIssue_NumberAndIssuedAtAgreeOnTheYear(t *testing.T) {
	s := newTestService()
	created := seedDraftInvoice(t, s)

	// 23:59:59 UTC on New Year's Eve is already the next year in Berlin.
	s.now = func() time.Time {
		return time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	issued, err := s.Issue(created.ID, testOwner)

	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)
	assert.Equal(t, "INV-2026-0001", *issued.InvoiceNumber)
	assert.Equal(t, 2026, issued.IssuedAt.Year())
}

func TestIssue_UsesTheNumberingOfTheOwner(t *testing.T) {
	repo := NewRepository()
	auckland, err := NewNumbering("RG", "Pacific/Auckland")
	require.NoError(t, err)
	repo.SetNumbering(testOwner, auckland)

	s := NewService(repo)
	mine := seedDraftInvoice(t, s)
	theirs, err := s.Create(draftInvoice(), otherOwner)
	require.NoError(t, err)

	s.now = func() time.Time {
		return time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC)
	}

	issued, err := s.Issue(mine.ID, testOwner)
	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)
	assert.Equal(t, "RG-2026-0001", *issued.InvoiceNumber)

	issuedOther, err := s.Issue(theirs.ID, otherOwner)
	require.NoError(t, err)
	require.NotNil(t, issuedOther.InvoiceNumber)
	assert.Equal(t, "INV-2025-0001", *issuedOther.InvoiceNumber,
		"ohne eigene Numbering gelten die Defaults, und der Zaehler laeuft getrennt")
}
