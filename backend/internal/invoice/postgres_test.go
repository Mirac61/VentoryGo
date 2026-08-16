package invoice

import (
	"context"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mirac61/VentoryGo/backend/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDSN(t *testing.T) string {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}

	// TEST_DATABASE_URL wins because these tests delete rows.
	for _, key := range []string{"TEST_DATABASE_URL", "DATABASE_URL"} {
		if dsn := os.Getenv(key); dsn != "" {
			return dsn
		}
	}
	t.Fatal("TEST_DATABASE_URL or DATABASE_URL must be set, or run with -short")
	return ""
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := db.NewPool(context.Background(), testDSN(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func createPostgresDraft(t *testing.T, s *Service, repo *PostgresRepository, owner string) Invoice {
	t.Helper()

	created, err := s.Create(draftInvoice(), owner)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(created.ID, owner) })
	return created
}

func readCounter(t *testing.T, pool *pgxpool.Pool, owner string, year int) int {
	t.Helper()

	var counter int
	err := pool.QueryRow(context.Background(),
		`SELECT counter FROM invoice_counters WHERE owner_id = $1 AND year = $2`, owner, year).Scan(&counter)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0
	}
	require.NoError(t, err)
	return counter
}

func counterOf(t *testing.T, number string) int {
	t.Helper()

	idx := strings.LastIndex(number, "-")
	require.Greaterf(t, idx, -1, "unexpected invoice number format %q", number)
	counter, err := strconv.Atoi(number[idx+1:])
	require.NoErrorf(t, err, "unexpected invoice number format %q", number)
	return counter
}

func yearOf(t *testing.T, number string) int {
	t.Helper()

	parts := strings.Split(number, "-")
	require.GreaterOrEqualf(t, len(parts), 2, "unexpected invoice number format %q", number)
	year, err := strconv.Atoi(parts[len(parts)-2])
	require.NoErrorf(t, err, "unexpected invoice number format %q", number)
	return year
}

func TestPostgresGetByID(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	owner := seedUser(t, pool)

	invoice := Invoice{
		ID:            uuid.NewString(),
		InvoiceNumber: new("INV-TEST-" + uuid.NewString()),
		Status:        StatusDraft,
		Sender:        Issuer{Contact: Contact{Name: "Sender", Street: "S1", Zip: "111", City: "C1", Country: "DE"}},
		Recipient:     Contact{Name: "Recipient", Street: "S2", Zip: "222", City: "C2", Country: "DE"},
		Items: []LineItem{
			{ID: uuid.NewString(), Position: 1, Description: "Test Item", Quantity: Quantity(1500), UnitPrice: 50, Total: 75, VatRate: 1900},
		},
	}

	created, err := repo.Create(invoice, owner)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(created.ID, owner) })

	fetched, err := repo.GetByID(created.ID, owner)
	require.NoError(t, err)
	require.NotNil(t, fetched.InvoiceNumber)
	assert.Equal(t, *created.InvoiceNumber, *fetched.InvoiceNumber)
	assert.Len(t, fetched.Items, 1)
	assert.Equal(t, "Test Item", fetched.Items[0].Description)
	assert.Equal(t, Quantity(1500), fetched.Items[0].Quantity)
	assert.Equal(t, Money(75), fetched.Items[0].Total)
}

func TestPostgresGetByID_DraftReturnsNilInvoiceNumber(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)

	created := createPostgresDraft(t, s, repo, owner)
	require.Nil(t, created.InvoiceNumber)

	fetched, err := repo.GetByID(created.ID, owner)

	require.NoError(t, err)
	assert.Nil(t, fetched.InvoiceNumber, `the empty column must read back as nil, not as a pointer to ""`)
}

func TestPostgresCreate_IgnoresClientSuppliedInvoiceNumber(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)

	const supplied = "INV-2026-9999"
	draft := draftInvoice()
	draft.InvoiceNumber = new(supplied)

	created, err := s.Create(draft, owner)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(created.ID, owner) })

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
	owner := seedUser(t, pool)

	const workers = 50
	const year = 2996

	drafts := make([]Invoice, 0, workers)
	for range workers {
		drafts = append(drafts, createPostgresDraft(t, s, repo, owner))
	}
	s.now = func() time.Time {
		return time.Date(year, 6, 1, 12, 0, 0, 0, DefaultNumbering().Location)
	}

	numbers := make([]string, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	for i, draft := range drafts {
		wg.Go(func() {
			issued, err := s.Issue(draft.ID, owner)
			if err != nil {
				errs[i] = err
				return
			}
			numbers[i] = *issued.InvoiceNumber
		})
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

	slices.Sort(counters)
	want := make([]int, 0, workers)
	for i := 1; i <= workers; i++ {
		want = append(want, i)
	}
	assert.Equal(t, want, counters, "the 50 issues must consume 1..50 without gaps")
}

func TestPostgresIssue_RollbackDoesNotConsumeNumber(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)

	const year = 2995

	failing := createPostgresDraft(t, s, repo, owner)
	next := createPostgresDraft(t, s, repo, owner)

	issuedAt := time.Date(year, 6, 1, 12, 0, 0, 0, DefaultNumbering().Location)
	s.now = func() time.Time { return issuedAt }

	boom := errors.New("boom")
	_, err := repo.Update(failing.ID, func(_ Invoice, _ Numbering, nextCounter func(time.Time) (int, error)) (Invoice, error) {
		if _, err := nextCounter(issuedAt); err != nil {
			return Invoice{}, err
		}
		return Invoice{}, boom
	}, owner)
	require.ErrorIs(t, err, boom)

	assert.Equal(t, 0, readCounter(t, pool, owner, year), "a rolled back issue must give the counter value back")

	unchanged, err := repo.GetByID(failing.ID, owner)
	require.NoError(t, err)
	assert.Equal(t, StatusDraft, unchanged.Status)
	assert.Nil(t, unchanged.InvoiceNumber)

	issued, err := s.Issue(next.ID, owner)
	require.NoError(t, err)
	require.NotNil(t, issued.InvoiceNumber)
	assert.Equal(t, 1, counterOf(t, *issued.InvoiceNumber), "the next issue gets the number the failed one drew")
}

func TestPostgresIssue_ResetsCounterOnNewYear(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)

	const oldYear, newYear = 2999, 3000

	last := createPostgresDraft(t, s, repo, owner)
	first := createPostgresDraft(t, s, repo, owner)

	s.now = func() time.Time {
		return time.Date(oldYear, 12, 31, 23, 59, 0, 0, DefaultNumbering().Location)
	}
	issuedLast, err := s.Issue(last.ID, owner)
	require.NoError(t, err)

	s.now = func() time.Time {
		return time.Date(newYear, 1, 1, 0, 1, 0, 0, DefaultNumbering().Location)
	}
	issuedFirst, err := s.Issue(first.ID, owner)
	require.NoError(t, err)

	require.NotNil(t, issuedLast.InvoiceNumber)
	require.NotNil(t, issuedFirst.InvoiceNumber)
	assert.Equal(t, "INV-2999-0001", *issuedLast.InvoiceNumber)
	assert.Equal(t, "INV-3000-0001", *issuedFirst.InvoiceNumber, "the new year starts over at 0001")

	// Read back from the database so the stored issued_at is what gets compared.
	for _, id := range []string{last.ID, first.ID} {
		stored, err := repo.GetByID(id, owner)
		require.NoError(t, err)
		require.NotNil(t, stored.InvoiceNumber)
		assert.Equalf(t, yearOf(t, *stored.InvoiceNumber), stored.IssuedAt.In(DefaultNumbering().Location).Year(),
			"number %q and issued_at must agree on the year", *stored.InvoiceNumber)
	}
}

func TestPostgresIssue_NewYearInConfiguredZoneWhileServerIsStillInOldYear(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)
	s := NewService(repo)
	owner := seedUser(t, pool)

	const oldYear, newYear = 2997, 2998

	draft := createPostgresDraft(t, s, repo, owner)

	// 23:30 UTC on New Year's Eve is already 00:30 on January 1st in Berlin.
	s.now = func() time.Time {
		return time.Date(oldYear, 12, 31, 23, 30, 0, 0, time.UTC)
	}
	issued, err := s.Issue(draft.ID, owner)
	require.NoError(t, err)

	require.NotNil(t, issued.InvoiceNumber)
	assert.Equal(t, "INV-2998-0001", *issued.InvoiceNumber)
	assert.Equal(t, 0, readCounter(t, pool, owner, oldYear), "the old year must not be touched")
	assert.Equal(t, 1, readCounter(t, pool, owner, newYear))
}
