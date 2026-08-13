package invoice

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type invoiceRepository interface {
	Create(invoice Invoice, ownerID string) (Invoice, error)
	GetByID(id string, ownerID string) (Invoice, error)
	GetAll(ownerID string) ([]Invoice, error)
	Delete(id string, ownerID string) error
	Update(id string, fn UpdateFunc, ownerID string) (Invoice, error)
}

type Service struct {
	repo invoiceRepository

	now func() time.Time
}

type MissingFieldsError struct {
	Fields []string
}

func NewService(repo invoiceRepository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func prepareItems(items []LineItem) {
	for i := range items {
		if items[i].ID == "" {
			items[i].ID = uuid.NewString()
		}
		items[i].Position = i + 1
		items[i].Total = Money(items[i].Quantity) * items[i].UnitPrice
	}
}

func calculateTotals(items []LineItem) (breakdown []VATBreakdownEntry, net, vat, gross Money) {
	prepareItems(items)
	grouped := make(map[int]Money)
	for _, item := range items {
		grouped[item.VatRate] += item.Total
	}
	rates := make([]int, 0, len(grouped))
	for rate := range grouped {
		rates = append(rates, rate)
	}
	sort.Ints(rates)
	for _, rate := range rates {
		netAmount := grouped[rate]
		vatAmount := RoundedVAT(netAmount, int64(rate))

		breakdown = append(breakdown, VATBreakdownEntry{
			VatRate:   rate,
			NetAmount: netAmount,
			VatAmount: vatAmount,
		})
		net += netAmount
		vat += vatAmount
	}
	gross = net + vat
	return
}

func validateInvoiceData(items []LineItem) error {
	if len(items) == 0 {
		return ErrInvalidInput
	}
	for _, item := range items {
		if item.Description == "" || item.Quantity <= 0 || item.UnitPrice < 0 || item.VatRate < 0 {
			return ErrInvalidInput
		}
	}
	return nil
}

func (e *MissingFieldsError) Error() string {
	return fmt.Sprintf("missing required fields: %s", strings.Join(e.Fields, ", "))
}

func validateForIssue(invoice Invoice) error {
	var missing []string

	if invoice.ServiceDate.IsZero() {
		missing = append(missing, "serviceDate")
	}

	if invoice.Currency == "" {
		missing = append(missing, "currency")
	}

	if invoice.Sender.IBAN == "" {
		missing = append(missing, "senderIban")
	}

	if invoice.Sender.VatID == "" && invoice.Sender.TaxNumber == "" {
		missing = append(missing, "senderVatId or senderTaxNumber")
	}

	if len(missing) > 0 {
		return &MissingFieldsError{Fields: missing}
	}
	return nil
}

func (s *Service) Create(invoice Invoice, ownerID string) (Invoice, error) {
	if ownerID == "" {
		return Invoice{}, ErrMissingOwner
	}
	if err := validateInvoiceData(invoice.Items); err != nil {
		return Invoice{}, err
	}
	if invoice.Currency == "" {
		invoice.Currency = "EUR"
	}

	invoice.ID = uuid.NewString()
	// Postgres TIMESTAMPTZ stores microsecond precision, so truncate here to
	// keep the in-memory value equal to what a later read from the DB returns.
	invoice.CreatedAt = s.now().Truncate(time.Microsecond)
	invoice.Status = StatusDraft
	invoice.InvoiceNumber = nil

	for i := range invoice.Items {
		invoice.Items[i].ID = uuid.NewString()
	}
	invoice.VatBreakdown, invoice.NetTotal, invoice.VATAmount, invoice.GrossTotal = calculateTotals(invoice.Items)

	return s.repo.Create(invoice, ownerID)
}

func fillBreakdown(inv *Invoice) {
	inv.VatBreakdown, inv.NetTotal, inv.VATAmount, inv.GrossTotal = calculateTotals(inv.Items)
}

func (s *Service) GetByID(id string, ownerID string) (Invoice, error) {
	if ownerID == "" {
		return Invoice{}, ErrMissingOwner
	}
	invoice, err := s.repo.GetByID(id, ownerID)
	if err != nil {
		return Invoice{}, err
	}
	fillBreakdown(&invoice)
	return invoice, nil
}

func (s *Service) GetAll(ownerID string) ([]Invoice, error) {
	if ownerID == "" {
		return nil, ErrMissingOwner
	}
	invoices, err := s.repo.GetAll(ownerID)
	if err != nil {
		return nil, err
	}
	for i := range invoices {
		fillBreakdown(&invoices[i])
	}
	return invoices, nil
}

func (s *Service) Delete(id string, ownerID string) error {
	if ownerID == "" {
		return ErrMissingOwner
	}
	invoice, err := s.repo.GetByID(id, ownerID)
	if err != nil {
		return err
	}
	if invoice.Status != StatusDraft {
		return ErrNotDeletable
	}
	return s.repo.Delete(id, ownerID)
}

func (s *Service) Update(id string, replacement Invoice, ownerID string) (Invoice, error) {
	if ownerID == "" {
		return Invoice{}, ErrMissingOwner
	}
	mutate := func(invoice Invoice, _ Numbering, _ func(time.Time) (int, error)) (Invoice, error) {
		if invoice.Status != StatusDraft {
			return Invoice{}, ErrNotUpdatable
		}

		replacement.ID = invoice.ID
		replacement.InvoiceNumber = invoice.InvoiceNumber
		replacement.Status = invoice.Status
		replacement.CreatedAt = invoice.CreatedAt
		replacement.IssuedAt = invoice.IssuedAt

		if replacement.Currency == "" {
			replacement.Currency = invoice.Currency
		}

		if err := validateInvoiceData(replacement.Items); err != nil {
			return Invoice{}, err
		}
		replacement.VatBreakdown, replacement.NetTotal, replacement.VATAmount, replacement.GrossTotal = calculateTotals(replacement.Items)
		return replacement, nil
	}
	return s.repo.Update(id, mutate, ownerID)
}

func (s *Service) PartialUpdate(id string, patch InvoicePatch, ownerID string) (Invoice, error) {
	if ownerID == "" {
		return Invoice{}, ErrMissingOwner
	}
	mutate := func(invoice Invoice, _ Numbering, _ func(time.Time) (int, error)) (Invoice, error) {
		if invoice.Status != StatusDraft {
			return Invoice{}, ErrNotUpdatable
		}

		if patch.Items != nil {
			invoice.Items = *patch.Items
		}
		if patch.Notes != nil {
			invoice.Notes = *patch.Notes
		}
		if patch.PaymentDueAt != nil {
			invoice.PaymentDueAt = *patch.PaymentDueAt
		}
		if patch.Recipient != nil {
			invoice.Recipient = *patch.Recipient
		}

		if err := validateInvoiceData(invoice.Items); err != nil {
			return Invoice{}, err
		}
		invoice.VatBreakdown, invoice.NetTotal, invoice.VATAmount, invoice.GrossTotal = calculateTotals(invoice.Items)
		return invoice, nil
	}
	return s.repo.Update(id, mutate, ownerID)
}

func (s *Service) Issue(id string, ownerID string) (Invoice, error) {
	if ownerID == "" {
		return Invoice{}, ErrMissingOwner
	}
	return s.repo.Update(id, func(invoice Invoice, numbering Numbering, nextCounter func(time.Time) (int, error)) (Invoice, error) {
		if invoice.Status != StatusDraft {
			return Invoice{}, ErrInvalidTransition
		}
		if err := validateForIssue(invoice); err != nil {
			return Invoice{}, err
		}

		// Nummer und issuedAt kommen aus demselben Zeitpunkt, sonst kann das
		// Jahr in der Nummer am Silvesterabend vom Jahr in issuedAt abweichen.
		now := s.now().In(numbering.Location).Truncate(time.Microsecond)
		counter, err := nextCounter(now)
		if err != nil {
			return Invoice{}, err
		}
		number := numbering.Format(now.Year(), counter)

		invoice.Status = StatusIssued
		invoice.IssuedAt = now
		invoice.InvoiceNumber = &number
		return invoice, nil
	}, ownerID)
}
