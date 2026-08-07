package invoice

import (
	"sync"
	"time"
)

type counterKey struct {
	ownerID string
	year    int
}

type Repository struct {
	invoices  map[string]Invoice
	numbering map[string]Numbering
	counters  map[counterKey]int
	mu        sync.RWMutex
	counterMu sync.Mutex
}

func NewRepository() *Repository {
	return &Repository{
		invoices:  make(map[string]Invoice),
		numbering: make(map[string]Numbering),
		counters:  make(map[counterKey]int),
	}
}

func (r *Repository) SetNumbering(ownerID string, numbering Numbering) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.numbering[ownerID] = numbering
}

func cloneInvoice(invoice Invoice) Invoice {
	invoice.Items = append([]LineItem(nil), invoice.Items...)
	return invoice
}

func (r *Repository) nextCounter(ownerID string, now time.Time) (int, error) {
	r.counterMu.Lock()
	defer r.counterMu.Unlock()

	key := counterKey{ownerID: ownerID, year: now.Year()}
	r.counters[key]++
	return r.counters[key], nil
}

// Ein leerer Owner wird abgewiesen, obwohl der Service ihn schon abfaengt:
// owner_id ist in Postgres NOT NULL mit Foreign Key, das In-Memory-Repo waere
// sonst nachsichtiger als die Datenbank, die es in Tests vertritt.
func (r *Repository) Create(invoice Invoice, ownerID string) (Invoice, error) {
	if ownerID == "" {
		return Invoice{}, ErrMissingOwner
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	invoice.OwnerID = ownerID
	stored := cloneInvoice(invoice)
	r.invoices[stored.ID] = stored
	return cloneInvoice(stored), nil
}

// Fremder Owner liefert ErrNotFound wie eine unbekannte ID: sonst waere an der
// Antwort ablesbar, dass die Rechnung existiert.
func (r *Repository) GetByID(id string, ownerID string) (Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	invoice, ok := r.invoices[id]
	if !ok || invoice.OwnerID != ownerID {
		return Invoice{}, ErrNotFound
	}
	return cloneInvoice(invoice), nil
}

func (r *Repository) GetAll(ownerID string) ([]Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Invoice, 0, len(r.invoices))
	for _, invoice := range r.invoices {
		if invoice.OwnerID != ownerID {
			continue
		}
		result = append(result, cloneInvoice(invoice))
	}
	return result, nil
}

func (r *Repository) Delete(id string, ownerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if invoice, ok := r.invoices[id]; !ok || invoice.OwnerID != ownerID {
		return ErrNotFound
	}
	delete(r.invoices, id)
	return nil
}

// UpdateFunc mutates an invoice. It runs inside the repository transaction so
// that read, modify and write happen atomically. nextCounter draws the next
// counter of the given year from the same transaction and must only be called
// when the number is actually used.
type UpdateFunc func(existing Invoice, numbering Numbering, nextCounter func(now time.Time) (int, error)) (Invoice, error)

func (r *Repository) Update(id string, fn UpdateFunc, ownerID string) (Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.invoices[id]
	if !ok || existing.OwnerID != ownerID {
		return Invoice{}, ErrNotFound
	}

	numbering, found := r.numbering[existing.OwnerID]
	if !found {
		numbering = DefaultNumbering()
	}

	nextCounter := func(now time.Time) (int, error) {
		return r.nextCounter(existing.OwnerID, now)
	}

	updated, err := fn(cloneInvoice(existing), numbering, nextCounter)
	if err != nil {
		return Invoice{}, err
	}
	updated.ID = existing.ID
	updated.OwnerID = existing.OwnerID

	stored := cloneInvoice(updated)
	r.invoices[id] = stored
	return cloneInvoice(stored), nil
}
