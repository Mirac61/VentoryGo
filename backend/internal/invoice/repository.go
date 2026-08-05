package invoice

import (
	"sync"
	"time"
)

type Repository struct {
	invoices    map[string]Invoice
	mu          sync.RWMutex
	counterMu   sync.Mutex
	counterYear int
	counter     int
}

func NewRepository() *Repository {
	return &Repository{
		invoices: make(map[string]Invoice),
	}
}

func cloneInvoice(invoice Invoice) Invoice {
	invoice.Items = append([]LineItem(nil), invoice.Items...)
	return invoice
}

func (r *Repository) nextCounter(now time.Time) (int, error) {
	r.counterMu.Lock()
	defer r.counterMu.Unlock()

	year := now.Year()
	if year != r.counterYear {
		r.counterYear = year
		r.counter = 0
	}
	r.counter++
	return r.counter, nil
}

func (r *Repository) Create(invoice Invoice, ownerID string) (Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	invoice.OwnerId = ownerID
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
	if !ok || invoice.OwnerId != ownerID {
		return Invoice{}, ErrNotFound
	}
	return cloneInvoice(invoice), nil
}

func (r *Repository) GetAll(ownerID string) ([]Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Invoice, 0, len(r.invoices))
	for _, invoice := range r.invoices {
		if invoice.OwnerId != ownerID {
			continue
		}
		result = append(result, cloneInvoice(invoice))
	}
	return result, nil
}

func (r *Repository) Delete(id string, ownerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if invoice, ok := r.invoices[id]; !ok || invoice.OwnerId != ownerID {
		return ErrNotFound
	}
	delete(r.invoices, id)
	return nil
}

// UpdateFunc mutates an invoice. It runs inside the repository transaction so
// that read, modify and write happen atomically. nextCounter draws the next
// counter of the given year from the same transaction and must only be called
// when the number is actually used.
type UpdateFunc func(existing Invoice, nextCounter func(now time.Time) (int, error)) (Invoice, error)

func (r *Repository) Update(id string, fn UpdateFunc, ownerID string) (Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.invoices[id]
	if !ok || existing.OwnerId != ownerID {
		return Invoice{}, ErrNotFound
	}

	updated, err := fn(cloneInvoice(existing), r.nextCounter)
	if err != nil {
		return Invoice{}, err
	}
	updated.ID = existing.ID
	updated.OwnerId = existing.OwnerId

	stored := cloneInvoice(updated)
	r.invoices[id] = stored
	return cloneInvoice(stored), nil
}
