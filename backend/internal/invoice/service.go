package invoice

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

type invoiceRepository interface {
	Create(invoice Invoice) (Invoice, error)
	GetByID(id string) (Invoice, error)
	GetAll() ([]Invoice, error)
	Delete(id string) error
	Update(id string, fn UpdateFunc) (Invoice, error)
}

type Service struct {
	repo      invoiceRepository
	numbering Numbering
	now       func() time.Time
}

type MissingFieldsError struct {
	Fields []string
}

func NewService(repo invoiceRepository) *Service {
	return NewServiceWithNumbering(repo, DefaultNumbering())
}

func NewServiceWithNumbering(repo invoiceRepository, numbering Numbering) *Service {
	return &Service{repo: repo, numbering: numbering, now: time.Now}
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

func calculateTotals(items []LineItem, vatRate float64) (net, vat, gross Money) {
	prepareItems(items)
	for _, item := range items {
		net += item.Total
	}
	ratePercent := int64(math.Round(vatRate * 100))
	vat = RoundedVAT(net, ratePercent)
	gross = net + vat
	return
}

func validateInvoiceData(items []LineItem, vatRate float64) error {
	if vatRate < 0 || vatRate > 1 {
		return ErrInvalidInput
	}
	if percent := vatRate * 100; math.Abs(percent-math.Round(percent)) > 1e-9 {
		return ErrInvalidInput
	}
	if len(items) == 0 {
		return ErrInvalidInput
	}
	for _, item := range items {
		if item.Description == "" || item.Quantity <= 0 || item.UnitPrice < 0 {
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

func (s *Service) Create(invoice Invoice) (Invoice, error) {
	if err := validateInvoiceData(invoice.Items, invoice.VATRate); err != nil {
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
	invoice.NetTotal, invoice.VATAmount, invoice.GrossTotal = calculateTotals(invoice.Items, invoice.VATRate)

	return s.repo.Create(invoice)
}

func (s *Service) GetByID(id string) (Invoice, error) {
	return s.repo.GetByID(id)
}

func (s *Service) GetAll() ([]Invoice, error) {
	return s.repo.GetAll()
}

func (s *Service) Delete(id string) error {
	invoice, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if invoice.Status != StatusDraft {
		return ErrNotDeletable
	}
	return s.repo.Delete(id)
}

func (s *Service) Update(id string, replacement Invoice) (Invoice, error) {
	mutate := func(invoice Invoice, _ func(time.Time) (int, error)) (Invoice, error) {
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

		if err := validateInvoiceData(replacement.Items, replacement.VATRate); err != nil {
			return Invoice{}, err
		}
		replacement.NetTotal, replacement.VATAmount, replacement.GrossTotal = calculateTotals(replacement.Items, replacement.VATRate)
		return replacement, nil
	}
	return s.repo.Update(id, mutate)
}

func (s *Service) PartialUpdate(id string, patch InvoicePatch) (Invoice, error) {
	mutate := func(invoice Invoice, _ func(time.Time) (int, error)) (Invoice, error) {
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
		if patch.VATRate != nil {
			invoice.VATRate = *patch.VATRate
		}

		if err := validateInvoiceData(invoice.Items, invoice.VATRate); err != nil {
			return Invoice{}, err
		}
		invoice.NetTotal, invoice.VATAmount, invoice.GrossTotal = calculateTotals(invoice.Items, invoice.VATRate)
		return invoice, nil
	}
	return s.repo.Update(id, mutate)
}

func (s *Service) Issue(id string) (Invoice, error) {
	return s.repo.Update(id, func(invoice Invoice, nextCounter func(time.Time) (int, error)) (Invoice, error) {
		if invoice.Status != StatusDraft {
			return Invoice{}, ErrInvalidTransition
		}
		if err := validateForIssue(invoice); err != nil {
			return Invoice{}, err
		}

		// Nummer und issuedAt kommen aus demselben Zeitpunkt, sonst kann das
		// Jahr in der Nummer am Silvesterabend vom Jahr in issuedAt abweichen.
		now := s.now().In(s.numbering.Location).Truncate(time.Microsecond)
		counter, err := nextCounter(now)
		if err != nil {
			return Invoice{}, err
		}
		number := s.numbering.Format(now.Year(), counter)

		invoice.Status = StatusIssued
		invoice.IssuedAt = now
		invoice.InvoiceNumber = &number
		return invoice, nil
	})
}
