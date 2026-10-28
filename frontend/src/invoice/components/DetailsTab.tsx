import { PlusIcon } from '@phosphor-icons/react'
import {
    CURRENCIES,
    VAT_RATES,
    formatDateGerman,
    formatVatRate,
    isRowSaveable,
    newRow,
    rowErrors,
    type DetailsForm,
    type ItemMode,
    type ItemRow,
} from '../detailsForm'
import { formatCentsDisplay } from '../money'
import { EditorField } from './EditorField'
import { EditorSelect } from './EditorSelect'
import { EditorTextArea } from './EditorTextArea'
import styles from './DetailsTab.module.css'

interface DetailsTabProps {
    details: DetailsForm
    invoiceNumber: string | null
    issuedDate: string
    vatExempt: boolean
    readOnly: boolean
    // Line totals as the backend calculated them, by item id (in cents).
    savedTotals: Map<string, number>
    onChange: (changed: Partial<DetailsForm>) => void
}

const MODE_OPTIONS = [
    { value: 'quantity', label: 'Menge × Preis' },
    { value: 'fixed', label: 'Festbetrag' },
]

export function DetailsTab({
    details,
    invoiceNumber,
    issuedDate,
    vatExempt,
    readOnly,
    savedTotals,
    onChange,
}: DetailsTabProps) {
    const { rows, currency } = details

    function updateRow(id: string, changes: Partial<ItemRow>) {
        onChange({ rows: rows.map((row) => (row.id === id ? { ...row, ...changes } : row)) })
    }

    function removeRow(id: string) {
        onChange({ rows: rows.filter((row) => row.id !== id) })
    }

    const currencyOptions = CURRENCIES.some((entry) => entry.value === currency)
        ? CURRENCIES
        : [...CURRENCIES, { value: currency, label: currency }]

    return (
        <>
            <section className={styles.card}>
                <h2 className={styles.cardTitle}>Rechnungsdetails</h2>
                <p className={styles.cardHelp}>Name, Datumsangaben und Währung der Rechnung.</p>

                <EditorField
                    label="Name der Rechnung"
                    hint="optional"
                    maxLength={200}
                    autoComplete="off"
                    value={details.title}
                    readOnly={readOnly}
                    onChange={(event) => onChange({ title: event.target.value })}
                />
                <EditorField
                    label="Rechnungs-Nr."
                    placeholder="Wird beim Ausstellen vergeben"
                    value={invoiceNumber ?? ''}
                    readOnly
                />
                <EditorField
                    label="Leistungsdatum"
                    type="date"
                    value={details.serviceDate}
                    readOnly={readOnly}
                    onChange={(event) => onChange({ serviceDate: event.target.value })}
                />
                <EditorField
                    label="Rechnungsdatum"
                    placeholder="Wird beim Ausstellen gesetzt"
                    value={formatDateGerman(issuedDate)}
                    readOnly
                />
                <EditorField
                    label="Fälligkeitsdatum"
                    type="date"
                    value={details.paymentDueDate}
                    readOnly={readOnly}
                    onChange={(event) => onChange({ paymentDueDate: event.target.value })}
                />
                <EditorSelect
                    label="Währung"
                    options={currencyOptions}
                    value={currency}
                    disabled={readOnly}
                    onChange={(event) => onChange({ currency: event.target.value })}
                />
                {vatExempt && (
                    <p className={styles.info}>Kleinunternehmen gemäß § 19 UStG: Es wird keine Umsatzsteuer berechnet.</p>
                )}
            </section>

            <section className={styles.card}>
                <h2 className={styles.cardTitle}>Positionen</h2>
                <p className={styles.cardHelp}>Leistungen und Beträge, die auf der Rechnung erscheinen.</p>

                {rows.map((row, index) => {
                    const errors = rowErrors(row)
                    const isFixed = row.mode === 'fixed'
                    // An incomplete row is not saved (the backend may hold a placeholder
                    // under its id), so it has no total of its own to show.
                    const total = isRowSaveable(row, vatExempt) ? savedTotals.get(row.id) : undefined
                    const rates = VAT_RATES.includes(row.vatRate) ? VAT_RATES : [...VAT_RATES, row.vatRate]

                    return (
                        <div key={row.id} className={styles.item}>
                            <div className={styles.itemHeader}>
                                <span className={styles.itemTitle}>Eintrag {index + 1}</span>
                                <button
                                    type="button"
                                    className={styles.removeButton}
                                    onClick={() => removeRow(row.id)}
                                    disabled={readOnly || rows.length === 1}
                                    title={rows.length === 1 ? 'Mindestens ein Eintrag ist nötig' : undefined}
                                >
                                    Eintrag entfernen
                                </button>
                            </div>

                            <EditorField
                                label="Leistung"
                                autoComplete="off"
                                value={row.description}
                                readOnly={readOnly}
                                onChange={(event) => updateRow(row.id, { description: event.target.value })}
                            />
                            <EditorSelect
                                label="Modus"
                                options={MODE_OPTIONS}
                                value={row.mode}
                                disabled={readOnly}
                                onChange={(event) => updateRow(row.id, { mode: event.target.value as ItemMode })}
                            />

                            {!isFixed && (
                                <div className={styles.grid2}>
                                    <EditorField
                                        label="Menge"
                                        inputMode="decimal"
                                        autoComplete="off"
                                        value={row.quantity}
                                        readOnly={readOnly}
                                        error={errors.quantity}
                                        onChange={(event) => updateRow(row.id, { quantity: event.target.value })}
                                    />
                                    <EditorField
                                        label="Einheit"
                                        hint="optional"
                                        placeholder="z. B. Std."
                                        autoComplete="off"
                                        value={row.unit}
                                        readOnly={readOnly}
                                        onChange={(event) => updateRow(row.id, { unit: event.target.value })}
                                    />
                                </div>
                            )}

                            <EditorField
                                label={isFixed ? 'Betrag' : 'Einzelpreis'}
                                inputMode="decimal"
                                placeholder="0,00"
                                autoComplete="off"
                                value={row.unitPrice}
                                readOnly={readOnly}
                                error={errors.unitPrice}
                                onChange={(event) => updateRow(row.id, { unitPrice: event.target.value })}
                            />
                            <EditorSelect
                                label="MwSt.-Satz"
                                options={rates.map((rate) => ({ value: String(rate), label: formatVatRate(rate) }))}
                                value={String(vatExempt ? 0 : row.vatRate)}
                                disabled={readOnly || vatExempt}
                                onChange={(event) => updateRow(row.id, { vatRate: Number(event.target.value) })}
                            />

                            <div className={styles.total}>
                                <span>Summe</span>
                                <span>{total === undefined ? '–' : formatCentsDisplay(total, currency)}</span>
                            </div>

                            {!isRowSaveable(row, vatExempt) && (
                                <p className={styles.rowNote}>
                                    Dieser Eintrag erscheint in der Vorschau, sobald Leistung und Beträge ausgefüllt sind.
                                </p>
                            )}
                        </div>
                    )
                })}

                <button
                    type="button"
                    className={styles.addButton}
                    onClick={() => onChange({ rows: [...rows, newRow(vatExempt)] })}
                    disabled={readOnly}
                >
                    <PlusIcon size={16} weight="bold" />
                    Weiterer Zeileneintrag
                </button>
            </section>

            <section className={styles.card}>
                <h2 className={styles.cardTitle}>Dokumentation</h2>
                <p className={styles.cardHelp}>Freier Text, der auf der Rechnung erscheint.</p>

                <EditorTextArea
                    label="Anmerkung"
                    value={details.notes}
                    readOnly={readOnly}
                    onChange={(event) => onChange({ notes: event.target.value })}
                />
            </section>
        </>
    )
}