BEGIN;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM invoice_items
    WHERE quantity % 1000 <> 0
  ) THEN
    RAISE EXCEPTION 'cannot roll back decimal quantities without data loss';
  END IF;
END $$;

ALTER TABLE invoice_items
ALTER COLUMN quantity TYPE INTEGER
USING quantity / 1000;

COMMIT;
