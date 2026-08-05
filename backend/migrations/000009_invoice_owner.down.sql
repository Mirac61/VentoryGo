BEGIN;

-- The rollback keeps the rows: unlike the up migration it may run at a point
-- where real invoices exist. Two owners may hold the same invoice number, which
-- the global index cannot express -- that case aborts with a readable message
-- instead of letting CREATE UNIQUE INDEX fail on an arbitrary duplicate.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM invoices
        WHERE invoice_number <> ''
        GROUP BY invoice_number
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'rollback blocked: invoice numbers are only unique per owner, the global index cannot be restored';
    END IF;
END $$;

DROP INDEX invoices_owner_id_invoice_number_key;

CREATE UNIQUE INDEX invoices_invoice_number_key
    ON invoices (invoice_number)
    WHERE invoice_number <> '';

DROP INDEX invoices_owner_id_created_at_idx;

ALTER TABLE invoices DROP COLUMN owner_id;

COMMIT;
