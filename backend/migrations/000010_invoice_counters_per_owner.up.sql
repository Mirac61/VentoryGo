BEGIN;

-- Die Zeilen tragen keinen Mandanten, aus dem sich owner_id ableiten liesse.
-- Ohne das DELETE begaenne der erste Mandant mitten in seiner Reihe statt bei
-- 0001. Bereits vergebene Nummern wuerde derselbe Zaehler danach ein zweites Mal
-- ausgeben, deshalb erst die Zusicherung, dass es keine gibt.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM invoices WHERE invoice_number <> '') THEN
        RAISE EXCEPTION 'migration blocked: issued invoices exist, resetting the counters would hand out their numbers twice';
    END IF;
END $$;

DELETE FROM invoice_counters;

ALTER TABLE invoice_counters
    ADD COLUMN owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE invoice_counters DROP CONSTRAINT invoice_counters_pkey;
ALTER TABLE invoice_counters ADD PRIMARY KEY (owner_id, year);

-- NOT NULL keeps out NULL but not '': an empty prefix would number invoices
-- "-2026-0001", and an empty zone silently resolves to UTC in Go.
ALTER TABLE users
    ADD COLUMN number_prefix TEXT NOT NULL DEFAULT 'INV'
        CONSTRAINT users_number_prefix_not_empty CHECK (number_prefix <> ''),
    ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Europe/Berlin'
        CONSTRAINT users_timezone_not_empty CHECK (timezone <> '');

COMMIT;
