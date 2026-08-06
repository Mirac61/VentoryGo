package invoice

import (
	"context"
	"slices"
	"sync"
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

	first := createPostgresDraft(t, s, repo, owner)
	second := createPostgresDraft(t, s, repo, stranger)

	s.now = func() time.Time {
		return time.Date(year, 6, 1, 12, 0, 0, 0, s.numbering.Location)
	}
	issued, err := s.Issue(first.ID, owner)
	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)

	issuedByStranger, err := s.Issue(second.ID, stranger)
	require.NoError(t, err, "dieselbe Nummer muss fuer einen anderen Owner erlaubt sein")
	require.NotNil(t, issuedByStranger.InvoiceNumber)
	require.Equal(t, *issued.InvoiceNumber, *issuedByStranger.InvoiceNumber)

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

func TestPostgresIssue_CounterRunsPerOwner(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)
	stranger := seedUser(t, pool)

	const year = 2993
	s.now = func() time.Time {
		return time.Date(year, 6, 1, 12, 0, 0, 0, s.numbering.Location)
	}

	// Abwechselnd ausstellen: aus einem gemeinsamen Zaehler bekaeme jede Reihe
	// nur jede zweite Nummer.
	for want := 1; want <= 3; want++ {
		for _, id := range []string{owner, stranger} {
			draft := createPostgresDraft(t, s, repo, id)
			issued, err := s.Issue(draft.ID, id)
			require.NoError(t, err)
			require.NotNil(t, issued.InvoiceNumber)
			assert.Equalf(t, want, counterOf(t, *issued.InvoiceNumber),
				"owner %s bekam %q", id, *issued.InvoiceNumber)
		}
	}

	assert.Equal(t, 3, readCounter(t, pool, owner, year))
	assert.Equal(t, 3, readCounter(t, pool, stranger, year))
}

func TestPostgresIssue_ConcurrentIssuesOfTwoOwnersStayInTheirOwnRange(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owners := []string{seedUser(t, pool), seedUser(t, pool)}

	const perOwner = 25
	const year = 2992
	s.now = func() time.Time {
		return time.Date(year, 6, 1, 12, 0, 0, 0, s.numbering.Location)
	}

	type issue struct {
		owner string
		id    string
	}
	work := make([]issue, 0, len(owners)*perOwner)
	for _, owner := range owners {
		for range perOwner {
			work = append(work, issue{owner: owner, id: createPostgresDraft(t, s, repo, owner).ID})
		}
	}

	numbers := make([]string, len(work))
	errs := make([]error, len(work))

	var wg sync.WaitGroup
	for i, w := range work {
		wg.Go(func() {
			issued, err := s.Issue(w.id, w.owner)
			if err != nil {
				errs[i] = err
				return
			}
			numbers[i] = *issued.InvoiceNumber
		})
	}
	wg.Wait()

	drawn := map[string][]int{}
	for i, err := range errs {
		require.NoErrorf(t, err, "issue %d fehlgeschlagen", i)
		drawn[work[i].owner] = append(drawn[work[i].owner], counterOf(t, numbers[i]))
	}

	want := make([]int, 0, perOwner)
	for i := 1; i <= perOwner; i++ {
		want = append(want, i)
	}
	for _, owner := range owners {
		counters := drawn[owner]
		slices.Sort(counters)
		assert.Equalf(t, want, counters, "owner %s muss 1..%d ohne Luecken ziehen", owner, perOwner)
	}
}
