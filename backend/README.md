# Invoice Backend

Go/Gin REST API für Rechnungsverwaltung, Postgres-Persistenz via `pgx`.

## Setup

```bash
cp .env.example .env   # DATABASE_URL, POSTGRES_USER/PASSWORD/DB ausfüllen
docker compose up -d postgres
migrate -path migrations -database "$DATABASE_URL" up
go run .
```

Server läuft auf `:8080`.

Ganzer Stack in Docker: `docker compose --profile docker up`. Dort migriert ein eigener
`migrate`-Service vor dem Backend-Start, der `migrate`-Aufruf von Hand entfällt.

Sonderzeichen im Passwort müssen in den `*_URL`-Variablen prozent-kodiert sein
(`@` → `%40`, `?` → `%3F`, …), sonst scheitert schon das Parsen des DSN.

## Tests

Die Postgres-Tests löschen Zeilen und laufen deshalb gegen eine eigene Datenbank.
Einmalig anlegen und migrieren:

```bash
docker compose exec postgres psql -U invoice_user -d invoice_db \
  -c "CREATE DATABASE invoice_test OWNER invoice_user;"
set -a; source .env; set +a   # DSN in die Shell, go test liest keine .env
migrate -path migrations -database "$TEST_DATABASE_URL" up
```

```bash
set -a; source .env; set +a   # in einer neuen Shell erneut nötig
go test ./...
go test -short ./...          # überspringt alles, was Postgres braucht

./test-api.sh      # Server muss laufen
./smoke_test.sh    # Server + Postgres müssen laufen
```

`TEST_DATABASE_URL` muss auf `invoice_test` zeigen: Ist die Variable leer, fallen die
Tests auf `DATABASE_URL` zurück und löschen in der Entwicklungs-Datenbank. Ist keine von
beiden gesetzt, schlagen die Postgres-Tests fehl, statt still zu skippen — vergessene
Konfiguration soll nicht wie ein grüner Lauf aussehen.

## API

Alle Endpunkte unter `/api/invoices`:

| Methode | Pfad | Beschreibung |
|---|---|---|
| POST | `/` | Rechnung anlegen (Status `draft`) |
| GET | `/` | Alle Rechnungen |
| GET | `/:id` | Einzelne Rechnung |
| PUT | `/:id` | Komplett ersetzen (Server-Felder wie `status`, `invoiceNumber` bleiben geschützt) |
| PATCH | `/:id` | Teilweise ändern |
| DELETE | `/:id` | Löschen (nur Drafts) |
| POST | `/:id/issue` | Rechnungsnummer vergeben, Status → `issued`, danach eingefroren |

## Geldbeträge: Cent als Integer

`unitPrice`, `total`, `netTotal`, `vatAmount` und `grossTotal` sind **Integer in Cent**,
nicht Euro mit Nachkommastellen:

```json
{ "unitPrice": 3333, "quantity": 3, "total": 9999 }
```

`3333` bedeutet `33,33 €`. `float64` kann Centbeträge nicht exakt darstellen (Rundungsfehler
summieren sich über mehrere Positionen) — für Rechnungen ist das inakzeptabel, daher
Ganzzahl-Cent (`type Money int64`, siehe `internal/invoice/money.go`).

`vatRate` bleibt ein Dezimalwert (`0.19` = 19 %), das ist ein Satz, kein Geldbetrag.

**Breaking Change fürs Frontend:** Beträge müssen beim Anzeigen durch 100 geteilt und mit
zwei Nachkommastellen formatiert werden, statt sie direkt als Euro-Float zu interpretieren.
