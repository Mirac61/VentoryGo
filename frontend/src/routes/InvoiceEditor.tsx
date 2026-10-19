import { useState } from 'react'
import { Link, Navigate, useParams } from 'react-router'
import { useInvoice } from '../invoice/useInvoice'
import { useInvoiceAutosave } from '../invoice/useInvoiceAutosave'
import { useInvoicePreview } from '../invoice/useInvoicePreview'
import { isPlausibleIban, isValidEmail, normalizeIban } from '../invoice/validation'
import { dateOnly, detailsFromInvoice, detailsPatch, type DetailsForm } from '../invoice/detailsForm'
import type { Contact, Invoice, Issuer } from '../invoice/Invoice'
import { CompanyCustomerTab } from '../invoice/components/CompanyCustomerTab'
import { DetailsTab } from '../invoice/components/DetailsTab'
import { EditorTopbar } from '../invoice/components/EditorTopbar'
import { PreviewPane } from '../invoice/components/PreviewPane'
import styles from './InvoiceEditor.module.css'

type TabId = 'company' | 'details'


function persistableEmail(email: string): string {
    const trimmed = email.trim()
    return trimmed === '' || isValidEmail(trimmed) ? trimmed : ''
}

function persistableSender(sender: Issuer): Issuer {
    const iban = normalizeIban(sender.iban)
    return { ...sender, email: persistableEmail(sender.email), iban: isPlausibleIban(iban) ? iban : '' }
}

function persistableRecipient(recipient: Contact): Contact {
    return { ...recipient, email: persistableEmail(recipient.email) }
}

export function InvoiceEditor() {
    const { id } = useParams()
    return id ? <InvoiceEditorLoader id={id} /> : <Navigate to="/dashboard" replace />
}

function InvoiceEditorLoader({ id }: { id: string }) {
    const { invoice, error } = useInvoice(id)

    if (error) {
        return (
            <div className={styles.message}>
                <p>Rechnung konnte nicht geladen werden.</p>
                <Link to="/dashboard">Zurück zum Dashboard</Link>
            </div>
        )
    }
    if (!invoice) return null

    return <EditorBody key={invoice.id} initial={invoice} />
}

function EditorBody({ initial }: { initial: Invoice }) {
    const [sender, setSender] = useState<Issuer>(initial.sender)
    const [recipient, setRecipient] = useState<Contact>(initial.recipient)
    const [details, setDetails] = useState<DetailsForm>(() => detailsFromInvoice(initial))
    const [tab, setTab] = useState<TabId>('company')
    const vatExempt = initial.vatExempt
 
    const [readOnly, setReadOnly] = useState(initial.status !== 'draft')

    const autosave = useInvoiceAutosave(initial.id, { onFrozen: () => setReadOnly(true) })
    const { pdfBytes, error: previewError } = useInvoicePreview(initial.id, autosave.revision)

    function changeSender(key: keyof Issuer, value: string) {
        const next: Issuer = { ...sender, [key]: value }
        setSender(next)
        autosave.save({ sender: persistableSender(next) })
    }

    function changeRecipient(key: keyof Contact, value: string) {
        const next: Contact = { ...recipient, [key]: value }
        setRecipient(next)
        autosave.save({ recipient: persistableRecipient(next) })
    }

    function changeDetails(changed: Partial<DetailsForm>) {
        const next: DetailsForm = { ...details, ...changed }
        setDetails(next)

        const patch = detailsPatch(changed, next, vatExempt)
        if (Object.keys(patch).length > 0) autosave.save(patch)
    }

    const savedItems = autosave.lastSaved?.items ?? initial.items
    const savedTotals = new Map<string, number>(savedItems.map((item): [string, number] => [item.id, item.total]))

    function openPreview() {
        if (!pdfBytes) return
        const url = URL.createObjectURL(new Blob([pdfBytes], { type: 'application/pdf' }))
        window.open(url, '_blank')
        window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
    }

    return (
        <div className={styles.page}>
            <EditorTopbar
                saveState={autosave.state}
                readOnly={readOnly}
                canPreview={pdfBytes !== null}
                onSave={() => void autosave.flushNow()}
                onPreview={openPreview}
            />

            <div className={styles.subbar}>
                <p className={styles.breadcrumb}>
                    <span>{readOnly ? 'Ausgestellt' : 'Entwurf'}</span> / <strong>Rechnung</strong>
                </p>
                <div className={styles.tabs} role="tablist" aria-label="Bearbeitungsschritte">
                    <button
                        type="button"
                        role="tab"
                        aria-selected={tab === 'company'}
                        className={tab === 'company' ? styles.tabActive : styles.tab}
                        onClick={() => setTab('company')}
                    >
                        Unternehmen &amp; Kunde
                    </button>
                    <button
                        type="button"
                        role="tab"
                        aria-selected={tab === 'details'}
                        className={tab === 'details' ? styles.tabActive : styles.tab}
                        onClick={() => setTab('details')}
                    >
                        Rechnungsdetails
                    </button>
                </div>
            </div>

            {readOnly && (
                <p className={styles.notice} role="status">
                    Diese Rechnung ist ausgestellt und kann nicht mehr bearbeitet werden.
                </p>
            )}

            <div className={styles.workspace}>
                <section className={styles.panel} aria-label="Rechnungsformular">
                    {tab === 'company' ? (
                        <CompanyCustomerTab
                            sender={sender}
                            recipient={recipient}
                            readOnly={readOnly}
                            onSenderChange={changeSender}
                            onRecipientChange={changeRecipient}
                        />
                    ) : (
                        <DetailsTab
                            details={details}
                            invoiceNumber={initial.invoiceNumber}
                            issuedDate={dateOnly(initial.issuedAt)}
                            vatExempt={vatExempt}
                            readOnly={readOnly}
                            savedTotals={savedTotals}
                            onChange={changeDetails}
                        />
                    )}
                </section>

                <PreviewPane pdfBytes={pdfBytes} error={previewError} />
            </div>
        </div>
    )
}