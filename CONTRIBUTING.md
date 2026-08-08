# Contributing

Kurzfassung dessen, was man wissen muss, um am Repo zu arbeiten. Details zur API und zu
den Geldbeträgen stehen im [Backend-README](backend/README.md).

## Setup

Zwei `.env`-Dateien, beide aus der jeweiligen `.env.example`:

```bash
cp .env.example .env                  # Postgres-Zugangsdaten für den Container
cp backend/.env.example backend/.env  # DSNs für go run / go test auf dem Host
```

Das Passwort muss in beiden Dateien identisch sein. Sonderzeichen in den `*_URL`-Variablen
prozent-kodieren (`@` → `%40`), sonst scheitert schon das Parsen des DSN.

Danach reicht ein Befehl — er startet Postgres, spielt die Migrationen ein und hängt
Backend und Frontend mit getrennt beschrifteter Ausgabe daneben. Strg-C beendet alles:

```bash
./scripts/dev.sh      # Linux, macOS
.\scripts\dev.ps1     # Windows (PowerShell 7)
```

Von Hand geht es weiterhin so:

```bash
docker compose up -d postgres
set -a; . ./backend/.env; set +a          # go run und migrate lesen keine .env
migrate -path backend/migrations -database "$DATABASE_URL" up
cd backend && go run .     # :8080
cd frontend && npm install && npm run dev
```

Ganzer Stack in Docker: `docker compose --profile docker up -d`. Dort übernimmt der
`migrate`-Service das Schema, der Aufruf von Hand entfällt.

Für die Test-Datenbank siehe [Tests](#tests) — die läuft nicht automatisch mit hoch.

## Struktur

```
.
├── backend/          Go + Gin + pgx
│   ├── migrations/   golang-migrate, .up.sql/.down.sql paarweise
│   └── internal/
└── frontend/         React + TypeScript + Vite
```

Neue Dateien folgen der Struktur, die schon da ist. Wenn unklar ist, wohin etwas gehört:
vorher fragen, nicht eine zweite Konvention danebenstellen.

## Branches und Commits

Branch vom aktuellen `main`, benannt nach der Art der Änderung:

```
feat/invoice-pdf-export
fix/vat-rounding
chore/bump-pgx
```

Commits nach [Conventional Commits](https://www.conventionalcommits.org/):
`feat`, `fix`, `chore`, `docs`, `refactor`, `test`. Betreff im Imperativ, klein, ohne
Punkt am Ende:

```
feat(invoice): add PDF export endpoint
fix(money): round VAT per line item, not on the total
```

## Pull Requests

- Gegen `main`, ein PR pro abgeschlossener Änderung
- Verlinktes Issue im Body (`Closes #42`), falls eines existiert
- CodeRabbit läuft automatisch; Anmerkungen entweder beheben oder begründet abhaken
- Erst mergen, wenn Build und Tests grün sind
- Squash-Merge, damit `main` eine Zeile pro Änderung behält

Bei größeren Umbauten vorher eine [Discussion](../../discussions) aufmachen. Spart es,
einen fertigen PR wieder auseinanderzunehmen.

## Codestil

Vor dem Push:

```bash
cd backend  && gofmt -l . && go vet ./...
cd frontend && npm run lint && npm run build
```

`npm run build` macht den Typecheck mit — ein grüner Lint allein reicht nicht.

Ansonsten:

- Bezeichner, Kommentare und Commit-Messages auf Englisch. Ausgenommen sind Testnamen und
  Assertion-Meldungen, die bleiben deutsch
- Kommentare nur, wenn das *Warum* nicht offensichtlich ist. Was der Code tut, steht im Code
- Keine Abkürzungen, keine Füllwörter wie `Manager` oder `Helper` im Namen
- Booleans als Prädikat: `isPaid`, `hasLineItems`
- Kein `use`-Präfix — auch nicht bei React-Hooks-nahen Modulen
- Keine Lösung bauen, die allgemeiner ist als das Problem

## Migrations

golang-migrate, immer als Paar:

```
000004_add_invoice_currency.up.sql
000004_add_invoice_currency.down.sql
```

Eine bereits gemergte Migration wird nicht mehr editiert — auch nicht bei einem Tippfehler.
Stattdessen kommt eine neue obendrauf. Wer eine angewandte Migration ändert, hat lokal ein
anderes Schema als alle anderen, und das fällt erst irgendwann später auf.

Schema-Änderungen brauchen einen Hinweis im PR, wenn das Frontend davon betroffen ist.

## Tests

Die Postgres-Tests löschen Zeilen und laufen deshalb gegen eine eigene Datenbank. Einmalig:

```bash
docker compose exec postgres psql -U invoice_user -d invoice_db \
  -c "CREATE DATABASE invoice_test OWNER invoice_user;"
set -a; source backend/.env; set +a
migrate -path backend/migrations -database "$TEST_DATABASE_URL" up
```

Dann:

```bash
set -a; source backend/.env; set +a   # go test liest keine .env
go test ./...
go test -short ./...                  # ohne alles, was Postgres braucht
```

`TEST_DATABASE_URL` muss auf `invoice_test` zeigen. Ist die Variable leer, fallen die Tests
auf `DATABASE_URL` zurück und räumen die Entwicklungs-Datenbank leer.

Neue Geschäftslogik kommt mit Tests. Bugfixes mit einem Test, der ohne den Fix rot ist —
sonst weiß niemand, ob der Fix den Bug trifft.

## Festgelegte Entscheidungen

Nicht nebenbei im PR umdrehen; wenn eine davon nicht mehr passt, gehört das in eine
Discussion:

| Entscheidung | Grund |
|---|---|
| `pgx` statt ORM | Explizites SQL, keine versteckten Queries |
| Geld als `int64` in Cent (`type Money`) | `float64` sammelt Rundungsfehler über Positionen |
| `vatRate` als Dezimalwert (`0.19`) | Ein Satz, kein Geldbetrag |
| UUIDs vom Client | Idempotente Requests ohne Round-Trip |
| Ausgestellte Rechnungen sind eingefroren | Nach `issue` zählt nur noch die Storno-Gutschrift |

## Issues vs. Discussions

Reproduzierbares Problem oder konkreter Task → Issue. Alles davor → Discussion.
