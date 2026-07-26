BEGIN;

ALTER TABLE invoices
    ADD COLUMN service_date    DATE NOT NULL DEFAULT '0001-01-01',
    ADD COLUMN currency        TEXT NOT NULL DEFAULT 'EUR',
    ADD COLUMN sender_vat_id     TEXT NOT NULL DEFAULT '',
    ADD COLUMN sender_tax_number TEXT NOT NULL DEFAULT '',
    ADD COLUMN sender_iban       TEXT NOT NULL DEFAULT '',
    ADD COLUMN sender_bic        TEXT NOT NULL DEFAULT '',
    ADD COLUMN sender_bank_name  TEXT NOT NULL DEFAULT '';

COMMIT;
