import type { Invoice } from './Invoice'

// Das Backend kennt vier Status (draft/issued/paid/cancelled). "Überfällig"
// ist keiner davon, sondern ergibt sich aus issued + paymentDueAt.
//
// Diese Ableitung ist bewusst die einzige im Frontend: Liste, Diagramm und
// Kennzahlen lesen alle hier, sonst heisst derselbe Zustand an drei Stellen
// anders ("Offen" in der Liste, "Gesendet" im Mockup, "Offene Rechnungen" in
// den Kacheln).
export type DisplayStatus = 'draft' | 'sent' | 'overdue' | 'paid' | 'cancelled'

// Reihenfolge im Diagramm und in der Legende: vom Unfertigen zum Abgeschlossenen.
export const STATUS_ORDER: DisplayStatus[] = ['draft', 'sent', 'overdue', 'paid', 'cancelled']

// Zustände, die das Backend heute erreichen kann (nur draft -> issued). Sie
// stehen auch dann in der Legende, wenn gerade keine Rechnung darin liegt --
// sonst springt die Liste, sobald die erste Rechnung eines Zustands entsteht.
// "paid" und "cancelled" fehlen hier absichtlich: es gibt keinen Endpunkt, der
// sie setzt. Sobald es ihn gibt, erscheinen sie automatisch, weil
// dashboardStats.ts jeden Zustand mit count > 0 mitnimmt.
export const REACHABLE_STATUSES: DisplayStatus[] = ['draft', 'sent', 'overdue']

export const STATUS_LABEL: Record<DisplayStatus, string> = {
    draft: 'Entwurf',
    sent: 'Gesendet',
    overdue: 'Überfällig',
    paid: 'Bezahlt',
    cancelled: 'Storniert',
}

// CSS-Variablen aus styles/tokens.css, damit Punkt, Balkensegment und Legende
// nicht auseinanderlaufen.
export const STATUS_COLOR: Record<DisplayStatus, string> = {
    draft: 'var(--invoice-draft)',
    sent: 'var(--invoice-sent-unpaid)',
    overdue: 'var(--invoice-overdue)',
    paid: 'var(--invoice-paid)',
    cancelled: 'var(--invoice-saved)',
}

// Nur der Kalendertag zählt: das Fälligkeitsdatum ist ein Datum, kein
// Zeitpunkt. Ein Vergleich der Zeitstempel würde eine Rechnung je nach Uhrzeit
// schon am Fälligkeitstag als überfällig zeigen. Sie bleibt "gesendet" bis
// zum Ende ihres Fälligkeitstags und wird erst am Tag danach überfällig.
function isoDatePart(value: string): string | null {
    const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value)
    return match ? match[0] : null
}

function localDatePart(date: Date): string {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
}

// `now` ist injizierbar, damit Tests nicht von der Uhr abhängen.
export function displayStatus(invoice: Invoice, now: Date = new Date()): DisplayStatus {
    if (invoice.status === 'issued') {
        const due = isoDatePart(invoice.paymentDueAt)
        // Unlesbares Datum: lieber "gesendet" als fälschlich "überfällig".
        if (due === null) return 'sent'
        return due < localDatePart(now) ? 'overdue' : 'sent'
    }
    return invoice.status
}
