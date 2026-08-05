package invoice

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllReturnsOnlyOwnInvoices(t *testing.T) {
	s := newTestService()

	for range 2 {
		_, err := s.Create(draftInvoice(), testOwner)
		require.NoError(t, err)
		_, err = s.Create(draftInvoice(), otherOwner)
		require.NoError(t, err)
	}

	mine, err := s.GetAll(testOwner)
	require.NoError(t, err)
	assert.Len(t, mine, 2)
	for _, invoice := range mine {
		assert.Equal(t, testOwner, invoice.OwnerID)
	}

	theirs, err := s.GetAll(otherOwner)
	require.NoError(t, err)
	assert.Len(t, theirs, 2)
}

func TestForeignInvoiceIsNotFound(t *testing.T) {
	s := newTestService()

	created, err := s.Create(draftInvoice(), testOwner)
	require.NoError(t, err)

	t.Run("GetByID", func(t *testing.T) {
		_, err := s.GetByID(created.ID, otherOwner)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Update", func(t *testing.T) {
		_, err := s.Update(created.ID, draftInvoice(), otherOwner)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		notes := "fremd"
		_, err := s.PartialUpdate(created.ID, InvoicePatch{Notes: &notes}, otherOwner)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Delete", func(t *testing.T) {
		assert.ErrorIs(t, s.Delete(created.ID, otherOwner), ErrNotFound)
	})

	t.Run("Issue", func(t *testing.T) {
		_, err := s.Issue(created.ID, otherOwner)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	// Nach allen Fehlversuchen muss die Rechnung unveraendert dastehen.
	after, err := s.GetByID(created.ID, testOwner)
	require.NoError(t, err)
	assert.Equal(t, StatusDraft, after.Status)
	assert.Equal(t, created.Notes, after.Notes)
}

// Ein leerer Owner ist ein Fehler im Aufrufer, keine leere Treffermenge: sonst
// laeuft die Abfrage ungefiltert ueber alle Mandanten.
func TestEmptyOwnerIsRejected(t *testing.T) {
	s := newTestService()

	created, err := s.Create(draftInvoice(), testOwner)
	require.NoError(t, err)

	_, err = s.Create(draftInvoice(), "")
	assert.ErrorIs(t, err, ErrMissingOwner)

	invoices, err := s.GetAll("")
	assert.ErrorIs(t, err, ErrMissingOwner)
	assert.Empty(t, invoices)

	_, err = s.GetByID(created.ID, "")
	assert.ErrorIs(t, err, ErrMissingOwner)

	_, err = s.Update(created.ID, draftInvoice(), "")
	assert.ErrorIs(t, err, ErrMissingOwner)

	notes := "leer"
	_, err = s.PartialUpdate(created.ID, InvoicePatch{Notes: &notes}, "")
	assert.ErrorIs(t, err, ErrMissingOwner)

	assert.ErrorIs(t, s.Delete(created.ID, ""), ErrMissingOwner)

	_, err = s.Issue(created.ID, "")
	assert.ErrorIs(t, err, ErrMissingOwner)
}

// OwnerID traegt json:"-", der Owner kann also nicht per Request gesetzt werden.
// Update rettet ihn zusaetzlich aus dem Bestand, damit auch ein anderer
// Schreibpfad ihn nicht ueberschreiben kann.
func TestUpdateKeepsOwner(t *testing.T) {
	s := newTestService()

	created, err := s.Create(draftInvoice(), testOwner)
	require.NoError(t, err)

	tampered := draftInvoice()
	tampered.OwnerID = otherOwner

	updated, err := s.Update(created.ID, tampered, testOwner)
	require.NoError(t, err)
	assert.Equal(t, testOwner, updated.OwnerID)

	stored, err := s.GetByID(created.ID, testOwner)
	require.NoError(t, err)
	assert.Equal(t, testOwner, stored.OwnerID)

	_, err = s.GetByID(created.ID, otherOwner)
	assert.ErrorIs(t, err, ErrNotFound, "die Rechnung darf nicht beim anderen Owner auftauchen")
}

func TestPostgresGetAllReturnsOnlyOwnInvoices(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)
	stranger := seedUser(t, pool)

	mine := createPostgresDraft(t, s, repo, owner)
	createPostgresDraft(t, s, repo, stranger)

	invoices, err := s.GetAll(owner)
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	assert.Equal(t, mine.ID, invoices[0].ID)
	assert.Equal(t, owner, invoices[0].OwnerID)
}

func TestPostgresForeignInvoiceIsNotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)
	stranger := seedUser(t, pool)

	created := createPostgresDraft(t, s, repo, owner)

	_, err := s.GetByID(created.ID, stranger)
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = s.Update(created.ID, draftInvoice(), stranger)
	assert.ErrorIs(t, err, ErrNotFound)

	assert.ErrorIs(t, s.Delete(created.ID, stranger), ErrNotFound)

	_, err = s.Issue(created.ID, stranger)
	assert.ErrorIs(t, err, ErrNotFound)

	// Direkt aus der Datenbank lesen: die Zeile muss den Fremdzugriff
	// unveraendert ueberstanden haben.
	var status string
	var ownerID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT status, owner_id FROM invoices WHERE id = $1`, created.ID).Scan(&status, &ownerID))
	assert.Equal(t, string(StatusDraft), status)
	assert.Equal(t, owner, ownerID)
}

// Der Unique-Index haengt seit der Migration an (owner_id, invoice_number).
func TestPostgresSameInvoiceNumberForDifferentOwners(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)
	stranger := seedUser(t, pool)

	const year = 2994
	claimCounterYears(t, pool, year)

	first := createPostgresDraft(t, s, repo, owner)
	second := createPostgresDraft(t, s, repo, stranger)

	// Beide Rechnungen ziehen aus demselben globalen Zaehler, weil der
	// Nummernkreis pro Mandant erst mit #50 kommt. Fuer diesen Test wird die
	// zweite Nummer deshalb direkt gesetzt.
	s.now = func() time.Time {
		return time.Date(year, 6, 1, 12, 0, 0, 0, s.numbering.Location)
	}
	issued, err := s.Issue(first.ID, owner)
	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)

	_, err = pool.Exec(context.Background(),
		`UPDATE invoices SET invoice_number = $1 WHERE id = $2`, *issued.InvoiceNumber, second.ID)
	require.NoError(t, err, "dieselbe Nummer muss fuer einen anderen Owner erlaubt sein")

	var count int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM invoices WHERE invoice_number = $1`, *issued.InvoiceNumber).Scan(&count))
	assert.Equal(t, 2, count)
}

func TestPostgresUpdateKeepsOwner(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)
	stranger := seedUser(t, pool)

	created := createPostgresDraft(t, s, repo, owner)

	tampered := draftInvoice()
	tampered.OwnerID = stranger
	_, err := s.Update(created.ID, tampered, owner)
	require.NoError(t, err)

	var stored string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT owner_id FROM invoices WHERE id = $1`, created.ID).Scan(&stored))
	assert.Equal(t, owner, stored, "owner_id darf von einem PUT nicht angefasst werden")
}
