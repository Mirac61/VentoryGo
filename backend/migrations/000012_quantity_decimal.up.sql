BEGIN;

-- Requires ACCESS EXCLUSIVE; run during a maintenance window.
ALTER TABLE invoice_items
ALTER COLUMN quantity TYPE BIGINT
USING quantity::BIGINT * 1000;

COMMIT;
