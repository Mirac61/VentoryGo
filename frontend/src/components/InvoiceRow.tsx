import type { Invoice } from '../invoice/Invoice'
import styles from './InvoiceRow.module.css'

interface InvoiceRowProps {
    invoice: Invoice
}

// Nur die 4 Status, die das Backend tatsaechlich kennt (draft/issued/paid/
// cancelled) -- das Redesign-Mockup zeigt 6 Labels (u.a. "Gespeichert",
// "Gesendet"), die noch keine Entsprechung im Datenmodell haben. Bis das
// geklaert ist, bleibt es bei diesen vier plus dem Ueberfaellig-Sonderfall,
// den dashboardStats.ts schon aus issued + paymentDueAt ableitet.
type DisplayStatus = 'draft' | 'open' | 'overdue' | 'paid' | 'cancelled'

function displayStatus(invoice: Invoice): DisplayStatus {
    if (invoice.status === 'issued') {
        return new Date(invoice.paymentDueAt) < new Date() ? 'overdue' : 'open'
    }
    return invoice.status
}

const STATUS_LABEL: Record<DisplayStatus, string> = {
    draft: 'Entwurf',
    open: 'Offen',
    overdue: 'Überfällig',
    paid: 'Bezahlt',
    cancelled: 'Storniert',
}

const STATUS_DOT_CLASS: Record<DisplayStatus, string> = {
    draft: styles.dotNeutral,
    open: styles.dotPrimary,
    overdue: styles.dotDanger,
    paid: styles.dotSuccess,
    cancelled: styles.dotNeutral,
}

function formatDate(iso: string): string {
    const date = new Date(iso)
    if (Number.isNaN(date.getTime())) return ''
    return date.toLocaleDateString('de-DE')
}

export function InvoiceRow({ invoice }: InvoiceRowProps) {
    const status = displayStatus(invoice)
    const number = invoice.invoiceNumber ?? 'Entwurf'

    return (
        <div className={styles.row}>
            <div className={styles.info}>
                <div className={styles.name}>{invoice.recipient.name}</div>
                <div className={styles.meta}>
                    {number}, Erstellt am {formatDate(invoice.createdAt)}
                </div>
            </div>
            <div className={styles.status}>
                <span className={`${styles.dot} ${STATUS_DOT_CLASS[status]}`} />
                {STATUS_LABEL[status]}
            </div>
        </div>
    )
}