import { useState } from 'react'
import { Link } from 'react-router'
import { TrashIcon } from '@phosphor-icons/react'
import type { Invoice } from '../invoice/Invoice'
import { STATUS_COLOR, STATUS_LABEL, displayStatus } from '../invoice/invoiceStatus'
import { formatIsoDate } from '../invoice/dates'
import { formatCentsDisplay } from '../invoice/money'
import styles from './InvoiceRow.module.css'

interface InvoiceRowProps {
    invoice: Invoice
    // Must not throw: the caller reports failures itself.
    onDelete: (id: string) => Promise<void>
}

// Statusableitung und Farben kommen aus invoiceStatus.ts -- dieselbe Quelle wie
// Diagramm und Kacheln im Dashboard. Vorher hiess `issued` hier "Offen" und im
// Mockup "Gesendet".
export function InvoiceRow({ invoice, onDelete }: InvoiceRowProps) {
    const [confirming, setConfirming] = useState(false)
    const [deleting, setDeleting] = useState(false)

    const status = displayStatus(invoice)
    const isDraft = invoice.status === 'draft'
    const name = invoice.title || invoice.recipient.name || 'Neuer Entwurf'
    const number = invoice.invoiceNumber ?? 'Noch keine Nummer'
    // Ein leerer Entwurf hat 0 Cent -- "0,00 €" waere hier eine Aussage,
    // die nicht stimmt, also lieber ein Strich.
    const amount =
        isDraft && invoice.grossTotal === 0
            ? '–'
            : formatCentsDisplay(invoice.grossTotal, invoice.currency)

    async function confirmDelete() {
        setDeleting(true)
        await onDelete(invoice.id)
        setDeleting(false)
        setConfirming(false)
    }

    return (
        <div className={styles.row}>
            <Link to={`/invoices/${invoice.id}`} className={styles.link}>
                <div className={styles.info}>
                    <div className={styles.name}>{name}</div>
                    <div className={styles.meta}>
                        {number} · Erstellt am {formatIsoDate(invoice.createdAt)}
                    </div>
                </div>
                <div className={styles.amount}>{amount}</div>
                <div className={styles.status}>
                    <span
                        className={styles.dot}
                        style={{ background: STATUS_COLOR[status] }}
                    />
                    {STATUS_LABEL[status]}
                </div>
            </Link>

            {/* Only drafts can be deleted, the backend answers 409 for anything else. */}
            {isDraft &&
                (confirming ? (
                    <div className={styles.confirm} role="group" aria-label="Löschen bestätigen">
                        <span>Entwurf löschen?</span>
                        <button
                            type="button"
                            className={styles.confirmCancel}
                            onClick={() => setConfirming(false)}
                            disabled={deleting}
                        >
                            Abbrechen
                        </button>
                        <button
                            type="button"
                            className={styles.confirmDelete}
                            onClick={() => void confirmDelete()}
                            disabled={deleting}
                        >
                            {deleting ? 'Löscht …' : 'Löschen'}
                        </button>
                    </div>
                ) : (
                    <button
                        type="button"
                        className={styles.deleteButton}
                        aria-label={`Entwurf „${name}“ löschen`}
                        title="Entwurf löschen"
                        onClick={() => setConfirming(true)}
                    >
                        <TrashIcon size={18} />
                    </button>
                ))}
        </div>
    )
}