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

## Passwort-Hashing begrenzen

Argon2id belegt pro Hash 19 MiB. Ohne Grenze startet jeder gleichzeitige Request einen
eigenen Hash, und der Speicher wächst linear mit der Zahl der Angreifer-Verbindungen.
`auth.Hasher` lässt deshalb nur eine feste Zahl gleichzeitig zu; wer keinen Platz bekommt,
wartet, statt abgelehnt zu werden. Bricht der Client vorher ab, wird gar nicht erst
gerechnet — der Platz wird über `ctx` freigegeben.

Der Default folgt `GOMAXPROCS` und damit dem CPU-Limit des Containers, nicht den Kernen des
Hosts (`runtime.NumCPU()` meldet in einem 1-CPU-Container weiterhin alle Host-Kerne).
`AUTH_HASH_CONCURRENCY` überschreibt das; ein ungültiger Wert bricht den Start ab. Beim
Start loggt der Server, worauf er sich eingestellt hat.

### Lasttest

```bash
set -a; source .env; set +a                      # DSN in die Shell, wie bei go test
../scripts/loadtest.sh                           # Default-Grenze
AUTH_HASH_CONCURRENCY=4 ../scripts/loadtest.sh   # engere Grenze
AUTH_HASH_CONCURRENCY=1000 ../scripts/loadtest.sh # praktisch unbegrenzt, Vergleichswert
```

**Der Lauf fährt die CPU über Minuten voll aus.** Die Maschine wird heiß, und ein Laptop
drosselt sich irgendwann selbst — ab da misst man die Drosselung, nicht den Server. Netzteil
anschließen, andere Programme schließen, zwischen Läufen abkühlen lassen. Der Test legt
zehntausende Nutzer an und löscht sie am Ende wieder, gehört also an eine Wegwerf-Datenbank.
Absolute Zahlen gelten nur für die Maschine, auf der sie entstanden sind.

Messung auf 12 Kernen (Apple Silicon), lokale Postgres, 12 s je Messpunkt, Server pro
Messpunkt neu gestartet. `RSS` ist die Hochwassermarke des Serverprozesses:

| Clients | ohne Grenze req/s / p50 / p99 / RSS | Default (12) | `AUTH_HASH_CONCURRENCY=4` |
|---|---|---|---|
| 4  | 182 / 22 ms / 26 ms / 216 MiB | 181 / 22 ms / 27 ms / 253 MiB | 181 / 22 ms / 25 ms / 253 MiB |
| 8  | 345 / 23 ms / 28 ms / 409 MiB | 342 / 23 ms / 29 ms / 447 MiB | 216 / 37 ms / 41 ms / 235 MiB |
| 16 | 417 / 38 ms / 58 ms / 677 MiB | 415 / 38 ms / 61 ms / 600 MiB | 216 / 74 ms / 80 ms / 216 MiB |
| 32 | 424 / 72 ms / 155 ms / 1134 MiB | 413 / 77 ms / 121 ms / 847 MiB | 213 / 149 ms / 161 ms / 255 MiB |
| 64 | 410 / 142 ms / 471 ms / 1611 MiB | 422 / 150 ms / 233 ms / 678 MiB | 216 / 296 ms / 309 ms / 256 MiB |

Ohne Grenze wächst der Speicher linear mit der Clientzahl (216 → 1611 MiB), während der
Durchsatz ab 16 Clients steht. Mit Grenze 4 bleibt das RSS über 4 → 64 Clients flach, die
Latenz steigt dafür linear — Requests warten in der Schlange, alle Antworten bleiben 201.
Der Default kostet keinen Durchsatz (422 vs. 410 req/s bei 64 Clients) und verbessert den
Tail (p99 471 → 233 ms), weil sich weniger Hashes um dieselbe Speicherbandbreite prügeln.

Die Baseline "ohne Grenze" ist der aktuelle Code mit sehr hohem Limit, kein Checkout des
alten Stands: verhaltensgleich, aber eine Simulation.

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

## Fehlerformat

Jede Fehlerantwort folgt einem einheitlichen Format:

```json
{ "code": "INVOICE_NOT_FOUND", "message": "invoice not found", "fields": {}, "requestId": "…" }
```

- `code` — stabiler Maschinencode (siehe Tabelle unten), niemals ein interner Fehlertext.
- `message` — anzeigbare Kurzbeschreibung.
- `fields` — bei Validierungsfehlern (422) Feldname → verletzte Regel, sonst leer.
- `requestId` — korreliert mit dem Log-Eintrag des Servers; bei 500ern den Fehlertext
  im Log suchen, nicht in der Antwort (dort steht nie SQL, `pq`-Fehler oder Tabellennamen).

| Status | Code | Bedeutung |
|---|---|---|
| 400 | `INVALID_BODY` | Body ist kein valides JSON |
| 401 | `UNAUTHENTICATED` | Kein oder ungültiger Session-Cookie |
| 401 | `INVALID_CREDENTIALS` | E-Mail oder Passwort falsch (kein Unterschied nach Ursache) |
| 401 | `SESSION_EXPIRED` | Session existiert nicht mehr |
| 404 | `USER_NOT_FOUND` | Nutzer nicht gefunden |
| 404 | `INVOICE_NOT_FOUND` | Rechnung nicht gefunden |
| 409 | `EMAIL_TAKEN` | E-Mail ist bereits registriert |
| 409 | `INVOICE_NOT_DELETABLE` | Nur Drafts sind löschbar |
| 409 | `INVOICE_NOT_EDITABLE` | Nur Drafts sind änderbar |
| 409 | `INVOICE_INVALID_TRANSITION` | Statuswechsel nicht erlaubt |
| 422 | `VALIDATION_FAILED` | Body verletzt Feldregeln, Details in `fields` |
| 422 | `INVOICE_INVALID` | Pflichtfelder fürs Anlegen fehlen |
| 422 | `INVOICE_INCOMPLETE` | Pflichtfelder fürs Ausstellen fehlen, Details in `fields` |
| 500 | `INTERNAL` | Interner Fehler, Details nur im Log mit `requestId` |

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

### Mengen in Rechnungspositionen

`quantity` ist eine JSON-Dezimalzahl mit höchstens drei Nachkommastellen, zum Beispiel
`1.5`, `0.75` oder `1.005`. Intern und in PostgreSQL wird die Menge als Ganzzahl mit der
Skalierung `1000` gespeichert: `1.5` wird zu `1500`. Dadurch werden Gleitkommafehler
vermieden. Die Positionssumme wird beim Zurückrechnen auf Cent genau einmal deterministisch
gerundet.
