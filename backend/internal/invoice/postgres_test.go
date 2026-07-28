package invoice

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mirac61/Invoice/backend/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDSN = "postgres://invoice_user:bevvyr-9fexki-qyhQup@localhost:5432/invoice_db?sslmode=disable"

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := db.NewPool(context.Background(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// createPostgresDraft stores a draft that is complete enough to be issued and
// removes it again when the test ends.
func createPostgresDraft(t *testing.T, s *Service, repo *PostgresRepository) Invoice {
	t.Helper()

	created, err := s.Create(draftInvoice())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(created.ID) })
	return created
}

// readCounter returns the counter the next issue of that year will build on.
// A missing row means no invoice has been issued for the year yet.
func readCounter(t *testing.T, pool *pgxpool.Pool, year int) int {
	t.Helper()

	var counter int
	err := pool.QueryRow(context.Background(),
		`SELECT counter FROM invoice_counters WHERE year = $1`, year).Scan(&counter)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0
	}
	require.NoError(t, err)
	return counter
}

// counterOf extracts the running number from a formatted invoice number. It
// reads the trailing segment so it keeps working once the format gains a
// prefix.
func counterOf(t *testing.T, number string) int {
	t.Helper()

	idx := strings.LastIndex(number, "-")
	require.Greaterf(t, idx, -1, "unexpected invoice number format %q", number)
	counter, err := strconv.Atoi(number[idx+1:])
	require.NoErrorf(t, err, "unexpected invoice number format %q", number)
	return counter
}

func TestPostgresGetByID(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)

	invoice := Invoice{
		ID:            uuid.NewString(),
		InvoiceNumber: ptr("INV-TEST-" + uuid.NewString()),
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
	require.NotNil(t, fetched.InvoiceNumber)
	assert.Equal(t, *created.InvoiceNumber, *fetched.InvoiceNumber)
	assert.Len(t, fetched.Items, 1)
	assert.Equal(t, "Test Item", fetched.Items[0].Description)
}

func TestPostgresGetByID_DraftReturnsNilInvoiceNumber(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)

	created := createPostgresDraft(t, s, repo)
	require.Nil(t, created.InvoiceNumber)

	fetched, err := repo.GetByID(created.ID)

	require.NoError(t, err)
	assert.Nil(t, fetched.InvoiceNumber, `the empty column must read back as nil, not as a pointer to ""`)
}

func TestPostgresCreate_IgnoresClientSuppliedInvoiceNumber(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)

	const supplied = "INV-2026-9999"
	draft := draftInvoice()
	draft.InvoiceNumber = ptr(supplied)

	created, err := s.Create(draft)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(created.ID) })

	assert.Nil(t, created.InvoiceNumber)

	var stored string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT invoice_number FROM invoices WHERE id = $1`, created.ID).Scan(&stored))
	assert.Equal(t, "", stored, "the column stays empty so the partial unique index keeps skipping drafts")

	var count int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM invoices WHERE invoice_number = $1`, supplied).Scan(&count))
	assert.Zero(t, count, "the supplied value must not appear anywhere in the table")
}

func TestPostgresIssue_ConcurrentDrawsDistinctNumbers(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)

	const workers = 50
	drafts := make([]Invoice, 0, workers)
	for range workers {
		drafts = append(drafts, createPostgresDraft(t, s, repo))
	}

	year := time.Now().Year()
	before := readCounter(t, pool, year)

	numbers := make([]string, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	for i, draft := range drafts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issued, err := s.Issue(draft.ID)
			if err != nil {
				errs[i] = err
				return
			}
			numbers[i] = *issued.InvoiceNumber
		}()
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "issue %d failed", i)
	}

	seen := make(map[string]bool, workers)
	counters := make([]int, 0, workers)
	for _, number := range numbers {
		require.Falsef(t, seen[number], "invoice number %q was handed out twice", number)
		seen[number] = true
		counters = append(counters, counterOf(t, number))
	}

	// The counter table is shared with everything else in the test database, so
	// the absolute values depend on earlier runs. What has to hold is that these
	// 50 issues consumed exactly the 50 slots following the counter read above:
	// no duplicates, no gaps.
	slices.Sort(counters)
	want := make([]int, 0, workers)
	for i := 1; i <= workers; i++ {
		want = append(want, before+i)
	}
	assert.Equal(t, want, counters)
}

func TestPostgresIssue_RollbackDoesNotConsumeNumber(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)

	failing := createPostgresDraft(t, s, repo)
	next := createPostgresDraft(t, s, repo)

	year := time.Now().Year()
	before := readCounter(t, pool, year)

	// Draw a number and then fail, the way a validation or write error after
	// nextNumber would.
	boom := errors.New("boom")
	_, err := repo.Update(failing.ID, func(_ Invoice, nextNumber func() (string, error)) (Invoice, error) {
		if _, err := nextNumber(); err != nil {
			return Invoice{}, err
		}
		return Invoice{}, boom
	})
	require.ErrorIs(t, err, boom)

	assert.Equal(t, before, readCounter(t, pool, year), "a rolled back issue must give the counter value back")

	unchanged, err := repo.GetByID(failing.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusDraft, unchanged.Status)
	assert.Nil(t, unchanged.InvoiceNumber)

	issued, err := s.Issue(next.ID)
	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)
	assert.Equal(t, before+1, counterOf(t, *issued.InvoiceNumber), "the next issue gets the number the failed one drew")
}
