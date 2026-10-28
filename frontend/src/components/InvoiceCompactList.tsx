import { Link } from 'react-router'
import { CaretRightIcon } from '@phosphor-icons/react'
import type { Invoice } from '../invoice/Invoice'
import { STATUS_LABEL, displayStatus, type DisplayStatus } from '../invoice/invoiceStatus'
import { formatIsoDate } from '../invoice/dates'
import { formatCentsDisplay } from '../invoice/money'
import styles from './InvoiceCompactList.module.css'

interface InvoiceCompactListProps {
    // Bereits sortiert -- die Sortierung gehoert zum Aufrufer, damit Liste und
    // Diagramm dieselbe Reihenfolge zeigen.
    invoices: Invoice[]
    loading: boolean
}

// Eine Zeile pro Rechnung, nach dem Muster der iOS-Speicherliste: Icon links,
// Name und Zustand, Betrag rechts. Bewusst ohne eigene Status-Spalte -- die
// Farbe des Icons und die Unterzeile tragen den Zustand, damit rechts nur ein
// Wert steht und die Zeile nicht zerfaellt.
function subtitle(invoice: Invoice, status: DisplayStatus): string {
    if (status === 'draft') {
        const created = formatIsoDate(invoice.createdAt)
        return created ? `${STATUS_LABEL[status]} · erstellt am ${created}` : STATUS_LABEL[status]
    }
    if (status === 'sent' || status === 'overdue') {
        const due = formatIsoDate(invoice.paymentDueAt)
        return due ? `${STATUS_LABEL[status]} · fällig am ${due}` : STATUS_LABEL[status]
    }
    return STATUS_LABEL[status]
}

export function InvoiceCompactList({ invoices, loading }: InvoiceCompactListProps) {
    if (loading) {
        return (
            <ul className={styles.list} aria-hidden="true">
                {[0, 1, 2].map((row) => (
                    <li key={row} className={styles.item}>
                        <div className={styles.row}>
                            <span className={styles.iconSkeleton} />
                            <span className={styles.body}>
                                <span className={styles.lineSkeleton} />
                                <span className={`${styles.lineSkeleton} ${styles.lineShort}`} />
                            </span>
                        </div>
                    </li>
                ))}
            </ul>
        )
    }

    if (invoices.length === 0) return null

    return (
        <ul className={styles.list}>
            {invoices.map((invoice) => {
                const status = displayStatus(invoice)
                const name = invoice.title || invoice.recipient.name || 'Neuer Entwurf'
                const amount =
                    invoice.status === 'draft' && invoice.grossTotal === 0
                        ? '–'
                        : formatCentsDisplay(invoice.grossTotal, invoice.currency)
                return (
                    <li key={invoice.id} className={styles.item}>
                        <Link
                            to={`/invoices/${invoice.id}`}
                            className={styles.row}
                            aria-label={`${name} öffnen`}
                        >
                            <span className={styles.body}>
                                <span className={styles.name}>{name}</span>
                                <span className={styles.meta}>{subtitle(invoice, status)}</span>
                            </span>
                            <span className={styles.amount}>{amount}</span>
                            <CaretRightIcon
                                size={14}
                                weight="bold"
                                className={styles.chevron}
                                aria-hidden="true"
                            />
                        </Link>
                    </li>
                )
            })}
        </ul>
    )
}
