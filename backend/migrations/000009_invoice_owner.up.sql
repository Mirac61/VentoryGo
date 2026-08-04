BEGIN;

-- No real invoices exist yet, so the rows are dropped instead of being pinned
-- to an artificial default user. invoice_items go with them via cascade.
DELETE FROM invoices;

ALTER TABLE invoices
    ADD COLUMN owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX invoices_owner_id_created_at_idx ON invoices (owner_id, created_at);

DROP INDEX invoices_invoice_number_key;

CREATE UNIQUE INDEX invoices_owner_id_invoice_number_key
    ON invoices (owner_id, invoice_number)
    WHERE invoice_number <> '';

COMMIT;
