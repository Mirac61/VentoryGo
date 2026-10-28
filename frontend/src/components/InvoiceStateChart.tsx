import type { ReactNode } from 'react'
import type { StateSummary } from '../invoice/dashboardStats'
import { STATUS_COLOR, STATUS_LABEL } from '../invoice/invoiceStatus'
import { formatCentsDisplay, formatCount } from '../invoice/money'
import styles from './InvoiceStateChart.module.css'

interface InvoiceStateChartProps {
    states: StateSummary[]
    totalCents: number
    totalCount: number
    currency: string
    loading: boolean
    // Die Rechnungsliste steht in derselben Karte unter der Legende -- so
    // gehoeren Balken und die konkreten Rechnungen sichtbar zusammen.
    children?: ReactNode
}

// Der Balken gewichtet nach Geld, nicht nach Anzahl: drei Entwuerfe ueber
// 100 EUR sind fuer das Dashboard weniger relevant als eine ueber 4.000 EUR.
// Sind alle Betraege 0 (leere Entwuerfe), faellt die Gewichtung auf die Anzahl
// zurueck, damit der Balken nicht leer bleibt.
function segmentWeight(entry: StateSummary, useMoney: boolean): number {
    return useMoney ? entry.cents : entry.count
}

function plural(count: number): string {
    return count === 1 ? 'Rechnung' : 'Rechnungen'
}

export function InvoiceStateChart({
    states,
    totalCents,
    totalCount,
    currency,
    loading,
    children,
}: InvoiceStateChartProps) {
    const useMoney = totalCents > 0

    if (loading) {
        return (
            <section className={styles.card} aria-busy="true">
                <div className={styles.header}>
                    <span className={styles.skeletonTotal} aria-hidden="true" />
                </div>
                <div className={`${styles.bar} ${styles.skeletonBar}`} aria-hidden="true" />
                <ul className={styles.legend} aria-hidden="true">
                    {[0, 1, 2].map((row) => (
                        <li key={row} className={styles.legendItem}>
                            <span className={styles.skeletonLegend} />
                        </li>
                    ))}
                </ul>
                <span className={styles.srOnly}>Rechnungsstatus wird geladen …</span>
                {children}
            </section>
        )
    }

    const summaryLabel = states
        .map(
            (entry) =>
                `${STATUS_LABEL[entry.status]}: ${formatCount(entry.count)} ${plural(entry.count)}, ` +
                formatCentsDisplay(entry.cents, currency),
        )
        .join('; ')

    return (
        <section className={styles.card}>
            <div className={styles.header}>
                <div className={styles.headerValue}>
                    <span className={styles.total}>{formatCentsDisplay(totalCents, currency)}</span>
                    <span className={styles.totalHint}>
                        {formatCount(totalCount)} {plural(totalCount)}
                    </span>
                </div>
            </div>

            {totalCount === 0 ? (
                <p className={styles.empty}>Noch keine Rechnungen erfasst.</p>
            ) : (
                <>
                    <div
                        className={styles.bar}
                        role="img"
                        aria-label={`Rechnungen nach Status. ${summaryLabel}.`}
                    >
                        {states
                            .filter((entry) => entry.count > 0)
                            .map((entry) => {
                                const weight = segmentWeight(entry, useMoney)
                                return (
                                    <span
                                        key={entry.status}
                                        className={styles.segment}
                                        style={{
                                            flexGrow: weight,
                                            // Ein Zustand mit Rechnungen, aber 0 EUR (leerer
                                            // Entwurf) soll trotzdem als schmaler Strich
                                            // sichtbar bleiben.
                                            minWidth: '6px',
                                            background: STATUS_COLOR[entry.status],
                                        }}
                                        title={`${STATUS_LABEL[entry.status]}: ${formatCentsDisplay(entry.cents, currency)}`}
                                    />
                                )
                            })}
                    </div>

                    <ul className={styles.legend}>
                        {states.map((entry) => {
                            const empty = entry.count === 0
                            return (
                                <li
                                    key={entry.status}
                                    className={`${styles.legendItem} ${empty ? styles.legendItemEmpty : ''}`}
                                >
                                    <span
                                        className={styles.dot}
                                        style={{ background: STATUS_COLOR[entry.status] }}
                                        aria-hidden="true"
                                    />
                                    <span className={styles.legendLabel}>{STATUS_LABEL[entry.status]}</span>
                                    <span className={styles.legendCount}>{formatCount(entry.count)}</span>
                                    <span className={styles.legendAmount}>
                                        {formatCentsDisplay(entry.cents, currency)}
                                    </span>
                                </li>
                            )
                        })}
                    </ul>
                </>
            )}

            {children}
        </section>
    )
}
