import { useState } from 'react'
import { Link } from 'react-router'
import { TrashIcon } from '@phosphor-icons/react'
import type { Invoice } from '../invoice/Invoice'
import styles from './InvoiceRow.module.css'

interface InvoiceRowProps {
    invoice: Invoice
    // Must not throw: the caller reports failures itself.
    onDelete: (id: string) => Promise<void>
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

export function InvoiceRow({ invoice, onDelete }: InvoiceRowProps) {
    const [confirming, setConfirming] = useState(false)
    const [deleting, setDeleting] = useState(false)

    const status = displayStatus(invoice)
    const isDraft = invoice.status === 'draft'
    const name = invoice.title || invoice.recipient.name || 'Neuer Entwurf'
    const number = invoice.invoiceNumber ?? 'Noch keine Nummer'

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
                        {number} · Erstellt am {formatDate(invoice.createdAt)}
                    </div>
                </div>
                <div className={styles.status}>
                    <span className={`${styles.dot} ${STATUS_DOT_CLASS[status]}`} />
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