package invoice

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const invoiceColumns = `
	id, invoice_number, status, created_at, issued_at, payment_due_at, service_date, currency, 
	sender_name, sender_street, sender_zip, sender_city, sender_country, sender_email, sender_phone, sender_tax_id, sender_vat_id, sender_tax_number, sender_iban, sender_bic, sender_bank_name,
	recipient_name, recipient_street, recipient_zip, recipient_city, recipient_country, recipient_email, recipient_phone, recipient_tax_id,
	vat_rate, net_total, vat_amount, gross_total, notes`

const itemColumns = `id, invoice_id, position, description, quantity, unit_price, unit, total`

type rowScanner interface {
	Scan(dest ...any) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func numberOrEmpty(n *string) string {
	if n == nil {
		return ""
	}
	return *n
}

func scanInvoice(row rowScanner) (Invoice, error) {
	var invoice Invoice
	var number string
	err := row.Scan(
		&invoice.ID, &number, &invoice.Status, &invoice.CreatedAt, &invoice.IssuedAt, &invoice.PaymentDueAt, &invoice.ServiceDate, &invoice.Currency,
		&invoice.Sender.Name, &invoice.Sender.Street, &invoice.Sender.Zip, &invoice.Sender.City, &invoice.Sender.Country, &invoice.Sender.Email, &invoice.Sender.Phone, &invoice.Sender.TaxID, &invoice.Sender.VatID, &invoice.Sender.TaxNumber, &invoice.Sender.IBAN, &invoice.Sender.BIC, &invoice.Sender.BankName,
		&invoice.Recipient.Name, &invoice.Recipient.Street, &invoice.Recipient.Zip, &invoice.Recipient.City, &invoice.Recipient.Country, &invoice.Recipient.Email, &invoice.Recipient.Phone, &invoice.Recipient.TaxID,
		&invoice.VATRate, &invoice.NetTotal, &invoice.VATAmount, &invoice.GrossTotal, &invoice.Notes,
	)
	if number != "" {
		invoice.InvoiceNumber = &number
	}
	return invoice, err
}

func scanItem(row rowScanner) (LineItem, error) {
	var item LineItem
	err := row.Scan(&item.ID, &item.InvoiceID, &item.Position, &item.Description, &item.Quantity, &item.UnitPrice, &item.Unit, &item.Total)
	return item, err
}

func loadItems(ctx context.Context, q queryer, invoiceID string) ([]LineItem, error) {
	rows, err := q.Query(ctx, `
		SELECT `+itemColumns+`
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY position
	`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]LineItem, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func insertItems(ctx context.Context, tx pgx.Tx, invoiceID string, items []LineItem) error {
	for _, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO invoice_items (`+itemColumns+`)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, item.ID, invoiceID, item.Position, item.Description, item.Quantity, item.UnitPrice, item.Unit, item.Total)
		if err != nil {
			return err
		}
	}
	return nil
}

// queryer covers both *pgxpool.Pool and pgx.Tx so the same read path works
// inside and outside a transaction.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *PostgresRepository) GetByID(id string) (Invoice, error) {
	ctx := context.Background()

	invoice, err := scanInvoice(r.pool.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM invoices WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}

	invoice.Items, err = loadItems(ctx, r.pool, invoice.ID)
	if err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

func (r *PostgresRepository) GetAll() ([]Invoice, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `SELECT `+invoiceColumns+` FROM invoices ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invoices := make([]Invoice, 0)
	for rows.Next() {
		invoice, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoice.Items = make([]LineItem, 0)
		invoices = append(invoices, invoice)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(invoices) == 0 {
		return invoices, nil
	}

	ids := make([]string, 0, len(invoices))
	index := make(map[string]int, len(invoices))
	for i, invoice := range invoices {
		ids = append(ids, invoice.ID)
		index[invoice.ID] = i
	}

	itemRows, err := r.pool.Query(ctx, `
		SELECT `+itemColumns+`
		FROM invoice_items
		WHERE invoice_id = ANY($1::uuid[])
		ORDER BY invoice_id, position
	`, ids)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		item, err := scanItem(itemRows)
		if err != nil {
			return nil, err
		}
		i, ok := index[item.InvoiceID]
		if !ok {
			continue
		}
		invoices[i].Items = append(invoices[i].Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	return invoices, nil
}

func (r *PostgresRepository) Create(invoice Invoice) (Invoice, error) {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback(ctx)

	if err := insertInvoice(ctx, tx, invoice); err != nil {
		return Invoice{}, err
	}
	for i := range invoice.Items {
		invoice.Items[i].InvoiceID = invoice.ID
	}
	if err := insertItems(ctx, tx, invoice.ID, invoice.Items); err != nil {
		return Invoice{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

func insertInvoice(ctx context.Context, tx pgx.Tx, invoice Invoice) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO invoices (`+invoiceColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34)
	`, invoice.ID, numberOrEmpty(invoice.InvoiceNumber), invoice.Status, invoice.CreatedAt, invoice.IssuedAt, invoice.PaymentDueAt, invoice.ServiceDate, invoice.Currency,
		invoice.Sender.Name, invoice.Sender.Street, invoice.Sender.Zip, invoice.Sender.City, invoice.Sender.Country, invoice.Sender.Email, invoice.Sender.Phone, invoice.Sender.TaxID, invoice.Sender.VatID, invoice.Sender.TaxNumber, invoice.Sender.IBAN, invoice.Sender.BIC, invoice.Sender.BankName,
		invoice.Recipient.Name, invoice.Recipient.Street, invoice.Recipient.Zip, invoice.Recipient.City, invoice.Recipient.Country, invoice.Recipient.Email, invoice.Recipient.Phone, invoice.Recipient.TaxID,
		invoice.VATRate, invoice.NetTotal, invoice.VATAmount, invoice.GrossTotal, invoice.Notes)
	return err
}

func (r *PostgresRepository) Delete(id string) error {
	ctx := context.Background()

	tag, err := r.pool.Exec(ctx, `DELETE FROM invoices WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nextInvoiceCounter(ctx context.Context, tx pgx.Tx, now time.Time) (int, error) {
	var counter int
	err := tx.QueryRow(ctx, `
		INSERT INTO invoice_counters (year, counter) VALUES ($1, 1)
		ON CONFLICT (year) DO UPDATE SET counter = invoice_counters.counter + 1
		RETURNING counter
	`, now.Year()).Scan(&counter)
	if err != nil {
		return 0, err
	}
	return counter, nil
}

func (r *PostgresRepository) Update(id string, fn UpdateFunc) (Invoice, error) {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback(ctx)

	existing, err := scanInvoice(tx.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM invoices WHERE id = $1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}

	existing.Items, err = loadItems(ctx, tx, existing.ID)
	if err != nil {
		return Invoice{}, err
	}

	nextCounter := func(now time.Time) (int, error) {
		return nextInvoiceCounter(ctx, tx, now)
	}

	updated, err := fn(existing, nextCounter)
	if err != nil {
		return Invoice{}, err
	}
	updated.ID = existing.ID

	_, err = tx.Exec(ctx, `
		UPDATE invoices SET
			invoice_number = $1, status = $2, issued_at = $3, payment_due_at = $4, service_date = $5, currency = $6,
			sender_name = $7, sender_street = $8, sender_zip = $9, sender_city = $10, sender_country = $11, sender_email = $12, sender_phone = $13, sender_tax_id = $14, sender_vat_id = $15, sender_tax_number = $16, sender_iban = $17, sender_bic = $18, sender_bank_name = $19,
			recipient_name = $20, recipient_street = $21, recipient_zip = $22, recipient_city = $23, recipient_country = $24, recipient_email = $25, recipient_phone = $26, recipient_tax_id = $27,
			vat_rate = $28, net_total = $29, vat_amount = $30, gross_total = $31, notes = $32
		WHERE id = $33
	`, numberOrEmpty(updated.InvoiceNumber), updated.Status, updated.IssuedAt, updated.PaymentDueAt, updated.ServiceDate, updated.Currency,
		updated.Sender.Name, updated.Sender.Street, updated.Sender.Zip, updated.Sender.City, updated.Sender.Country, updated.Sender.Email, updated.Sender.Phone, updated.Sender.TaxID, updated.Sender.VatID, updated.Sender.TaxNumber, updated.Sender.IBAN, updated.Sender.BIC, updated.Sender.BankName,
		updated.Recipient.Name, updated.Recipient.Street, updated.Recipient.Zip, updated.Recipient.City, updated.Recipient.Country, updated.Recipient.Email, updated.Recipient.Phone, updated.Recipient.TaxID,
		updated.VATRate, updated.NetTotal, updated.VATAmount, updated.GrossTotal, updated.Notes, updated.ID)
	if err != nil {
		return Invoice{}, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM invoice_items WHERE invoice_id = $1`, updated.ID); err != nil {
		return Invoice{}, err
	}
	for i := range updated.Items {
		if updated.Items[i].ID == "" {
			updated.Items[i].ID = uuid.NewString()
		}
		updated.Items[i].InvoiceID = updated.ID
	}
	if err := insertItems(ctx, tx, updated.ID, updated.Items); err != nil {
		return Invoice{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return updated, nil
}
