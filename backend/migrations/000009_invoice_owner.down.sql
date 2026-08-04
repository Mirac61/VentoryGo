BEGIN;

-- Two owners may share an invoice number, which the global index cannot hold.
-- The rollback drops the rows, mirroring the up migration.
DELETE FROM invoices;

DROP INDEX invoices_owner_id_invoice_number_key;

CREATE UNIQUE INDEX invoices_invoice_number_key
    ON invoices (invoice_number)
    WHERE invoice_number <> '';

DROP INDEX invoices_owner_id_created_at_idx;

ALTER TABLE invoices DROP COLUMN owner_id;

COMMIT;
