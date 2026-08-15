BEGIN;

-- vat_rate wandert von Rechnung auf Position, und wird von
-- NUMERIC(5,4) (0.1900) auf INTEGER-Basispunkte (1900) umgestellt.

ALTER TABLE invoice_items ADD COLUMN vat_rate INTEGER NOT NULL DEFAULT 0;

UPDATE invoice_items ii
SET vat_rate = ROUND(i.vat_rate * 10000)::INTEGER
FROM invoices i
WHERE ii.invoice_id = i.id;

ALTER TABLE invoice_items ALTER COLUMN vat_rate DROP DEFAULT;

ALTER TABLE invoices DROP COLUMN vat_rate;

COMMIT;
