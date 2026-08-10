BEGIN;

-- Zwei Mandanten koennen denselben Zaehlerstand fuer ein Jahr halten, was ein
-- globaler Zaehler nicht ausdruecken kann. Statt willkuerlich eine Zeile zu
-- gewinnen, bricht der Rollback lesbar ab.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM invoice_counters
        GROUP BY year
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'rollback blocked: invoice counters are only unique per owner, the global counter cannot be restored';
    END IF;
END $$;

ALTER TABLE users
    DROP COLUMN timezone,
    DROP COLUMN number_prefix;

ALTER TABLE invoice_counters DROP COLUMN owner_id;
ALTER TABLE invoice_counters ADD PRIMARY KEY (year);

COMMIT;
