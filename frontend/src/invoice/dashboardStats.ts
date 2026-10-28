import { useMemo } from 'react'
import type { Invoice } from './Invoice'
import { REACHABLE_STATUSES, STATUS_ORDER, displayStatus, type DisplayStatus } from './invoiceStatus'

export interface StateSummary {
    status: DisplayStatus
    count: number
    // Summe der Bruttobetraege in Cent. Nicht mischen mit Euro-Werten.
    cents: number
}

export interface DashboardStats {
    // Alle erreichbaren Zustaende, auch leere, plus jeder weitere Zustand, in
    // dem tatsaechlich Rechnungen liegen -- in STATUS_ORDER.
    states: StateSummary[]
    totalCents: number
    totalCount: number
    // Alle Rechnungen tragen ihre eigene Waehrung, das Backend erlaubt jede
    // dreibuchstabige. Eine Umrechnung gibt es nicht -- die Anzeige nimmt die
    // Waehrung der ersten Rechnung und summiert alles. Solange nur EUR
    // vorkommt, ist das korrekt; bei gemischten Waehrungen waere die Summe
    // falsch und muesste pro Waehrung gruppiert werden.
    currency: string
}

export function useDashboardStats(invoices: Invoice[]): DashboardStats {
    return useMemo(() => buildDashboardStats(invoices), [invoices])
}

export function buildDashboardStats(invoices: Invoice[], now: Date = new Date()): DashboardStats {
    const buckets = new Map<DisplayStatus, StateSummary>(
        STATUS_ORDER.map((status) => [status, { status, count: 0, cents: 0 }]),
    )

    let totalCents = 0

    for (const invoice of invoices) {
        const bucket = buckets.get(displayStatus(invoice, now))!
        bucket.count++
        bucket.cents += invoice.grossTotal
        totalCents += invoice.grossTotal
    }

    const states = STATUS_ORDER.map((status) => buckets.get(status)!).filter(
        (entry) => entry.count > 0 || REACHABLE_STATUSES.includes(entry.status),
    )

    return {
        states,
        totalCents,
        totalCount: invoices.length,
        currency: invoices[0]?.currency ?? 'EUR',
    }
}
