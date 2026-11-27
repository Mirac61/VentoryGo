# Architecture

System boundaries, schema, and key flows. Library and protocol decisions live in [`docs/adr/`](adr/README.md).

## System Overview

Single Go binary + PostgreSQL 16 + React 19 SPA. Pending SQL migrations (`backend/migrations/*.sql`) are embedded in the binary and run automatically before `:8080` opens (`MIGRATE_ON_START=false` disables this).

```mermaid
flowchart LR
  Browser["Browser"] --> Proxy["Reverse / Dev Proxy (:8090)"]
  Proxy -->|"/, /landing/*"| Landing["Astro Landing (:4173)"]
  Proxy -->|"/login, /dashboard, /invoices"| SPA["React 19 + Vite (:5173)"]
  SPA -->|"/api/*"| API["Go / Gin API (:8080)"]
  API --> Auth["internal/auth"]
  API --> Invoice["internal/invoice + PDF"]
  Auth --> DB[("PostgreSQL 16")]
  Invoice --> DB
```

## Backend Structure

```text
backend/
├── main.go / router.go / migrate.go   Wiring, middleware, embedded migrations
└── internal/
    ├── auth/                          Users, Argon2id hasher, sessions, RequireAuth
    ├── invoice/                       Domain model, money/quantity math, numbering, CRUD
    │   └── pdf/                       DIN 5008 Form B generator (Maroto v2 + embedded fonts)
    ├── db/                            pgxpool setup
    └── httperror/                     Request ID + uniform JSON error writer
```

> [!IMPORTANT]
> `internal/auth` and `internal/invoice` never import each other. `router.go` connects them via `user_id` in the Gin context.

| Layer | Responsibility | Details |
|---|---|---|
| **Handler** | HTTP binding & errors | Maps domain errors to `{ code, message, fields, requestId }`; SQL details stay in server logs. |
| **Service** | Business rules | Cent totals, basis-point VAT grouping, § 19 UStG rules, draft-only edit/delete guards. |
| **Repository** | Persistence (`pgx`) | Explicit SQL behind a Go interface, paired with an in-memory implementation for fast tests. |

## Database Schema

```mermaid
erDiagram
  users ||--o{ sessions : "owns"
  users ||--o{ invoices : "owns"
  users ||--o{ invoice_counters : "sequences"
  invoices ||--|{ invoice_items : "contains"

  users {
    uuid id PK
    text email UK "lower(email)"
    text password_hash
    text number_prefix "default INV"
    text timezone "default Europe/Berlin"
    timestamptz created_at
  }

  sessions {
    text token_hash PK "SHA-256"
    uuid user_id FK
    timestamptz created_at
    timestamptz expires_at
  }

  invoices {
    uuid id PK
    uuid owner_id FK
    text invoice_number "unique per owner where <> ''"
    text status "draft | issued | paid | cancelled"
    text title
    date service_date
    timestamptz payment_due_at
    text currency "default EUR"
    boolean vat_exempt
    bigint net_total "cents"
    bigint vat_amount "cents"
    bigint gross_total "cents"
    text sender_and_recipient "inline address & tax snapshot"
  }

  invoice_items {
    uuid id PK
    uuid invoice_id FK
    integer position
    text description
    bigint quantity "scaled x1000"
    bigint unit_price "cents"
    integer vat_rate "basis points"
    bigint total "cents"
  }

  invoice_counters {
    uuid owner_id PK
    integer year PK
    integer counter
  }
```

| Choice | Reason |
|---|---|
| **Inline addresses on `invoices`** | Issued invoices are legal snapshots; future address edits must never alter past invoices. |
| **Fixed-point integers** | `BIGINT` cents, `INTEGER` basis points (`1900` = 19 %), and `BIGINT` quantities (`×1000`) avoid float drift. |
| **Partial unique index** | `(owner_id, invoice_number) WHERE invoice_number <> ''` allows unlimited drafts (`''`) while enforcing unique numbers once issued. |

## Security & Tenant Isolation

| Mechanism | How it works |
|---|---|
| **Sessions ([ADR 0001](adr/0001-authentifizierung.md))** | `HttpOnly` cookie holds 32 random bytes; PostgreSQL stores only `SHA-256(token)`. |
| **Throttled renewal** | `RequireAuth` extends `expires_at` at most once every 15 minutes (`min(15m, ttl/2)`), avoiding a write per request. |
| **Bounded Argon2id** | Semaphore caps concurrent 19 MiB hashes to `GOMAXPROCS` (`AUTH_HASH_CONCURRENCY`); unknown emails run a dummy hash. |
| **Query-level scoping** | Every query filters by `WHERE id = $1 AND owner_id = $2`; foreign IDs return `404`, never `403`. |

## Issuing an Invoice

`POST /api/invoices/:id/issue` locks the draft, allocates the next gap-free number, and freezes the record in one transaction:

```mermaid
sequenceDiagram
  participant C as Client
  participant S as invoice.Service
  participant DB as PostgreSQL (tx)

  C->>S: POST /api/invoices/:id/issue
  S->>DB: BEGIN + SELECT ... FOR UPDATE
  S->>DB: Load user prefix & timezone
  S->>S: Validate draft & required fields
  S->>DB: UPSERT invoice_counters RETURNING counter
  S->>DB: UPDATE invoices (status='issued', number)
  S->>DB: COMMIT
  S-->>C: 200 OK (frozen Invoice)
```

- **No double-issuing:** `FOR UPDATE` serializes parallel calls on the same draft.
- **No skipped numbers:** Counter increments *after* validation inside the transaction; any failure rolls it back.
- **Timezone-accurate year:** Computed in `users.timezone` (`Europe/Berlin`), so New Year's Eve stays in the right tax year.
- **Immutable afterward:** `PUT`, `PATCH`, and `DELETE` reject non-drafts with `409`.

## Editor Autosave & Live PDF

```mermaid
flowchart LR
  Edit["Form edit"] -->|"500 ms debounce"| Save["PATCH /api/invoices/:id"]
  Save -->|"revision++"| Preview["GET /api/invoices/:id/pdf"]
  Preview -->|"ArrayBuffer"| Canvas["pdfjs-dist Canvas"]
```

- **Serialized queue (`useInvoiceAutosave`):** One `PATCH` runs at a time; edits typed mid-flight merge into the next request, and unmount flushes pending changes.
- **Partial recovery on `422`:** If a combined patch fails validation, the hook retries field by field so valid fields still save.
- **Stale-free preview (`useInvoicePreview`):** Cancels slower in-flight PDF fetches via `AbortController`; Maroto v2 ([ADR 0002](adr/0002-pdf-bibliothek.md)) derives PDF timestamps from the invoice for byte-identical output.
