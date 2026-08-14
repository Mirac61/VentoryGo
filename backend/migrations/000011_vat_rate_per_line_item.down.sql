BEGIN;

-- Rollback: vat_rate zurueck auf Rechnungsebene, als NUMERIC(5,4).
-- Gemischte Saetze pro Rechnung gehen verloren -- der kleinste Satz
-- wird als Rechnungssatz uebernommen (best-effort).

ALTER TABLE invoices ADD COLUMN vat_rate NUMERIC(5,4) NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM invoice_items
        GROUP BY invoice_id
        HAVING COUNT(DISTINCT vat_rate) > 1
    ) THEN
        RAISE EXCEPTION 'rollback blocked: mixed VAT rates per invoice exist';
    END IF;
END $$;

UPDATE invoices i
SET vat_rate = (
    SELECT MIN(ii.vat_rate / 10000.0)
    FROM invoice_items ii
    WHERE ii.invoice_id = i.id
);

ALTER TABLE invoices ALTER COLUMN vat_rate DROP DEFAULT;

ALTER TABLE invoice_items DROP COLUMN vat_rate;

COMMIT;
