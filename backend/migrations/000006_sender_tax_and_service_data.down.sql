BEGIN;

ALTER TABLE invoices
    DROP COLUMN service_date,
    DROP COLUMN currency,
    DROP COLUMN sender_vat_id,
    DROP COLUMN sender_tax_number,
    DROP COLUMN sender_iban,
    DROP COLUMN sender_bic,
    DROP COLUMN sender_bank_name;

COMMIT;
