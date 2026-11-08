# Frontend code review

- **Date:** 2026-11-04
- **Scope:** `frontend/src` (React app). `landing/` was not reviewed.
- **Status:** deferred. The current branch (`feat/103/landing-page-migration-auf-astro-und-inhalte-korrigieren`) is landing-page only; apply these changes after it merges.
- **Load assumed:** one user, tens to hundreds of invoices per list.
- **Line numbers** refer to the working tree at this date. They may move after the landing branch merges.

**What this frontend does:** A React 19 + Vite app for a Go invoice API. It handles login/register, a dashboard with an invoice list and a status chart, and an invoice editor with autosave and a live PDF preview.

## Must fix

### 1. Every user is greeted as "Max"

`frontend/src/routes/Dashboard.tsx:17,75-77`

- **What this is:** The heading is `Servus, Max!`, with `Max` from the constant `FIRST_NAME_PLACEHOLDER`.
- **Problem:** The user model has no name, so `anna@firma.de` also sees "Servus, Max!". The first page after login states a wrong fact.
- **Fix:** Remove the name (`Servus!`) or derive one from `user.email.split('@')[0]`. One line, no new data.
- **If we skip it:** Users think they opened someone else's account.

### 2. The last edit is lost when the tab closes

`frontend/src/invoice/useInvoiceAutosave.ts:154-169`

- **What this is:** Edits wait 500 ms in a timer before the PATCH. In-app navigation flushes that timer; closing or reloading the tab does not.
- **Problem:** Type one word and press Ctrl+W within 500 ms: React does not unmount, the cleanup never runs, and the browser cancels a normal `fetch` on unload. The edit disappears without a message.
- **Fix:** Add a `pagehide` listener that calls `flushNow()`, and send that PATCH with `keepalive: true` (`ApiOptions` already passes through `RequestInit`). About five lines.
- **If we skip it:** Autosave loses exactly the edits users make right before leaving.

## Should fix

### 3. The dashboard shows every invoice twice

`frontend/src/routes/Dashboard.tsx:94-96,117-121`; `frontend/src/components/InvoiceCompactList.tsx`

- **What this is:** The left column lists all invoices; the right card renders the newest five again.
- **Problem:** With 20 invoices the screen renders 25 rows, and the same invoice appears twice. `InvoiceCompactList` plus its CSS (~213 lines) exists only for this copy. The name and empty-draft amount rules are also copied in `InvoiceRow.tsx:25-32` and `InvoiceCompactList.tsx:63-67`, so a change needs two edits.
- **Fix:** Delete `InvoiceCompactList` and the `children` prop of `InvoiceStateChart` — the full list is already on screen. If the card must keep a list, make the chart segments filter the left list. If both stay, move `invoiceName`/`invoiceAmount` into one helper.
- **If we skip it:** Duplicate rows and two places to keep in sync.

### 4. Four copies of the same error handling

`frontend/src/invoice/useInvoiceAutosave.ts:47-49`; `useInvoices.ts:38-41`; `useInvoice.ts:19-21`; `useInvoicePreview.ts:18-21`

- **What this is:** All four hooks convert `unknown` to `Error` with the same expression, and three repeat cancel/loading/error state.
- **Problem:** The next change to error reporting — logging, preserving `ApiError.status`, better user text — needs four edits. One of them will be missed.
- **Fix:** Export `toError(caught)` from `auth/api.ts` and use it in all four. Leave the hooks separate; the blob, refetch and revision cases differ too much for one generic hook.
- **If we skip it:** Errors behave differently per screen.

### 5. Two owners for date formatting

`frontend/src/invoice/detailsForm.ts:74-82`; `frontend/src/invoice/dates.ts:12-29`; `frontend/src/routes/InvoiceEditor.tsx:176`; `frontend/src/invoice/components/DetailsTab.tsx:90`

- **What this is:** `detailsForm` exports `dateOnly` and `formatDateGerman`; `dates.ts` exports `formatIsoDate`, which does the same job and also validates real calendar dates.
- **Problem:** The invoice date goes through `dateOnly` → `formatDateGerman`; every other date goes through `formatIsoDate`. The two handle a bad value differently: one would print `31.02.2026`, the other returns `''`.
- **Fix:** Pass the raw `initial.issuedAt` to `DetailsTab` and use `formatIsoDate`; delete `formatDateGerman`; move `dateOnly` into `dates.ts`. About ten lines net removed.
- **If we skip it:** The next date display picks one of two functions, and they can disagree.

### 6. Four field components repeat the same accessibility markup

`frontend/src/auth/components/Field.tsx:11-35`; `frontend/src/invoice/components/EditorField.tsx:10-38`; `EditorSelect.tsx:14-32`; `EditorTextArea.tsx:9-22`

- **What this is:** All four call `useId`, render a label with hint, wire `aria-describedby`/`aria-invalid`, and render an error message.
- **Problem:** A fix such as `aria-errormessage` or a different error id must be applied four times. The auth field already shows an error icon; the editor fields do not.
- **Fix:** Add one small `FieldShell` that renders the wrapper, label, hint and error and takes `id`, `error`, `hint`, `children`. The four components keep their own input type and CSS.
- **If we skip it:** The forms drift, and the next accessibility fix is done twice.

### 7. Eleven disabled "coming soon" controls

`frontend/src/components/Sidebar.tsx:18-23,84-94`; `login.tsx:65-89`; `register.tsx:90-101,135-144`; `CompanyCustomerTab.tsx:55-58`; `EditorTopbar.tsx:66-72,78-81`

- **What this is:** Three navigation items, forgot-password, GitHub login on both pages, the name field, logo upload, undo/redo, and "Rechnung ausstellen" are rendered but can never work.
- **Problem:** They look like broken features. Two appear on the first screen after opening the app. Each carries markup, icon imports and CSS until the feature lands.
- **Fix:** Delete them now; add each back in the change that adds its handler. If the layout needs them, use one shared disabled style instead of eleven bespoke comments.
- **If we skip it:** Users click dead controls and report bugs.

### 8. Dead and repeated CSS, plus unused images

`frontend/src/components/Navbar.module.css:18-84`; `frontend/src/auth/components/AuthCard.module.css:32-37`; `frontend/src/auth/components/authForm.module.css:68-81`; `frontend/src/routes/InvoiceEditor.module.css:93-98`; `frontend/src/styles/tokens.css:2,63-73`; `frontend/src/assets/`

- **What this is:** `Navbar` renders only a title, but its CSS still has a full account dropdown, duplicating the Radix menu in `Sidebar.module.css`. `.headerLine`, `.forgotPassword`, `.placeholder`, the `pulse` keyframes (defined twice) and three images are unused. `body` sets the sans font, yet 59 rules repeat `font-family: var(--font-sans)`. The `.dark` theme is never applied.
- **Problem:** These selectors cannot match anything. The duplicate dropdown and keyframes will be copied into the next component; the dark block suggests dark mode works when it does not.
- **Fix:** Delete the dead rules and images. Add `button, input, select, textarea { font: inherit; }` to `globals.css`, then remove the repeated `font-family` lines. Delete the `.dark` block and dark import until a toggle exists.
- **If we skip it:** Dead weight and bad examples for the next contributor.

### 9. The register page shows the login headline

`frontend/src/register/register.tsx:81`; `frontend/src/auth/components/AuthCard.tsx:7`

- **What this is:** `Register` passes `title="Melde dich an, um fortzufahren"`. `AuthCard` also declares a `subtitle` prop that is never rendered or passed.
- **Problem:** A new user is asked to log in on the page that creates the account. The unused prop is a leftover from the design.
- **Fix:** Use "Erstelle dein Konto" and remove `subtitle` from `AuthCardProps`.
- **If we skip it:** The first screen for a new user is confusing.

### 10. Clearing a date is silently ignored

`frontend/src/invoice/detailsForm.ts:194-206`; `frontend/src/invoice/components/DetailsTab.tsx:80-99`

- **What this is:** `detailsPatch` skips an empty `serviceDate`/`paymentDueDate` because the backend rejects zero dates (`notzero`), but the form still shows the empty value.
- **Problem:** Clear "Fälligkeitsdatum", wait for "Gespeichert", reload: the old date is back. The UI showed a blank field that was never saved.
- **Fix:** Do not let the field go empty: keep the previous date in `DetailsTab` when the new value is `''`, or show "Datum kann nicht geleert werden". The backend has no clear path, so the UI should not pretend.
- **If we skip it:** Users think they deleted a date and it returns later.

### 11. No frontend tests for money, dates, status or autosave

`frontend/package.json` scripts

- **What this is:** The frontend has dev/build/lint/preview but no test script and no test file.
- **Problem:** `parseCents`/`saveableItems` decide what is written; `displayStatus` decides overdue; the 422 retry decides what is dropped. A small change can break money or data with no failing test.
- **Fix:** Add Vitest (Vite is already there) and three focused tests: parsing, `buildDashboardStats` with mixed currencies and one overdue invoice, and the autosave 422 retry. One file.
- **If we skip it:** The riskiest logic is only checked by clicking.

## Nice to have

### 12. The site root sends signed-in users to the login form

`frontend/src/App.tsx:32-34`; `frontend/src/login/login.tsx:9-35`

- **What this is:** The catch-all route always navigates to `/login`, and `Login` does not redirect an already signed-in user.
- **Problem:** A logged-in user who opens `/` or an old link sees the login form although `/dashboard` works. It looks like the session was lost.
- **Fix:** Route `/` to `<Navigate to="/dashboard" replace />` and let `RequireAuth` handle anonymous users. One line.
- **If we skip it:** Bookmarking the root produces a confusing dead end.

### 13. The chart builds the same status text twice

`frontend/src/components/InvoiceStateChart.tsx:68-75,122-126`

- **What this is:** `summaryLabel` for the screen reader and each segment `title` repeat the same `mixedCurrencies ? … : …` expression.
- **Problem:** Changing the wording means editing two expressions that must produce identical text.
- **Fix:** Add one `statusText(entry)` helper and use it in both places. About six lines removed.
- **If we skip it:** The tooltip and the screen-reader label can disagree.

### 14. The same fetch error is printed twice

`frontend/src/routes/Dashboard.tsx:87,114`; `frontend/src/components/InvoiceStateChart.tsx:93-96`

- **What this is:** When the first load fails and there are no cached invoices, the list column shows `Fehler: …` and the chart shows the same string.
- **Problem:** The user sees the same failure twice on one screen, in two places.
- **Fix:** Render the fetch error once — for example only in the chart — and let the list show its empty state.
- **If we skip it:** Duplicate noise and two places to update error copy.

### 15. Every successful field retry refetches the PDF

`frontend/src/invoice/useInvoiceAutosave.ts:71-81,103-130`; `frontend/src/invoice/useInvoicePreview.ts:8-24`

- **What this is:** `patchOnce` increments `revision` after every PATCH; the preview fetches `/pdf` on every revision.
- **Problem:** A PATCH with one invalid field and five valid fields becomes up to six PATCHes. Each successful one triggers a server-side PDF render, so the preview redraws several times for one save.
- **Fix:** Count the whole `drain` as one revision: keep the last saved invoice in a local variable and call `setRevision` once after the loop.
- **If we skip it:** Saves feel slow on large invoices or weak machines.

## Verdict

Fix 1 and 2 first. Then 3, 7 and 11 are the biggest wins.

**Lean:** Roughly -350 lines possible: about -213 for the second dashboard list, -100 in dead/repeated CSS, and -40 in stale stubs.

**Not checked:** `landing/`, the Go API beyond the response shapes I read, and no runtime check because `frontend/node_modules` is not installed.
