BEGIN;

-- Die Zeilen tragen keinen Mandanten, aus dem sich owner_id ableiten liesse, und
-- Migration 9 hat die zugehoerigen Rechnungen bereits geloescht. Ohne das DELETE
-- begaenne der erste Mandant mitten in seiner Reihe statt bei 0001.
DELETE FROM invoice_counters;

ALTER TABLE invoice_counters
    ADD COLUMN owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE invoice_counters DROP CONSTRAINT invoice_counters_pkey;
ALTER TABLE invoice_counters ADD PRIMARY KEY (owner_id, year);

-- Gelesen werden die Spalten erst, wenn Numbering pro Mandant gebaut wird.
ALTER TABLE users
    ADD COLUMN number_prefix TEXT NOT NULL DEFAULT 'INV',
    ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Europe/Berlin';

COMMIT;
