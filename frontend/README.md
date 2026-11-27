# VentoryGo Frontend

React 19 + TypeScript + Vite + Tailwind CSS v4 single-page client for the [Backend API](../backend/README.md).

## Setup & Scripts

```bash
npm ci
npm run dev       # Vite dev server on :5173 (proxies /api -> localhost:8080)
npm run lint      # Oxlint
npm run build     # TypeScript check (tsc -b) + production bundle in dist/
npm run preview   # Serve production build locally
```

> [!NOTE]
> `vite.config.ts` sets `strictPort: true` on `:5173` (expected by `dev-proxy/proxy.js`) and allows `../shared/fonts` via `server.fs.allow`.

## Routes & Structure

| Route | Component | Access |
|---|---|---|
| `/login`, `/register` | `login/`, `register/` | Public |
| `/dashboard` | `routes/Dashboard.tsx` | `RequireAuth` |
| `/invoices/:id` | `routes/InvoiceEditor.tsx` | `RequireAuth` |
| `*` | Redirect → `/login` | Fallback |

| Directory | Contents |
|---|---|
| `src/auth/` | `AuthProvider`, typed `api` / `apiBlob` fetch wrappers, localized error mapping |
| `src/invoice/` | Invoice types, money/date helpers, dashboard stats, editor tabs, autosave & preview hooks |
| `src/components/` | Shared UI (`Sidebar`, `Navbar`, `InvoiceRow`, `PdfCanvas`) |
| `src/styles/` | Design tokens (`tokens.css`) and global styles |

## Key Mechanics

| Module | Behavior |
|---|---|
| **`AuthProvider`** | Hydrates `/api/auth/me` on mount using a generation counter so a slow initial `401` cannot overwrite a newer login. Any mid-session `401` clears state and redirects to `/login`. Separates account creation from post-register login failures via `PostRegisterLoginError`. |
| **`useInvoiceAutosave`** | Debounces edits (500 ms) into a single-flight `PATCH` queue. Retries field by field on `422` so valid fields still save, flushes pending edits on unmount, and locks the editor on `409` (issued elsewhere). |
| **`useInvoicePreview`** | Watches `(invoiceId, revision)` and fetches `GET /api/invoices/:id/pdf` with an `AbortController` so slow responses never overwrite newer renders. |
| **`PdfCanvas`** | Renders PDF bytes via `pdfjs-dist`, scales to container width (`ResizeObserver`), caps canvas area at `32 MP` to avoid high-DPI browser crashes, and swaps pages atomically via `replaceChildren` without flicker. |

Money values from the API are **integer cents** (`3333` = `33,33 €`) and VAT rates are **basis points** (`1900` = `19 %`); see [`src/invoice/money.ts`](src/invoice/money.ts) and [`backend/README.md`](../backend/README.md#geldbeträge-cent-als-integer).
