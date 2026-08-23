import { useMemo } from 'react'
import type { Invoice } from './Invoice'

interface DashboardStats {
    entwuerfe: number
    offeneRechnungen: number
    ueberfaellig: number
    bezahltDiesenMonat: number
}


export function useDashboardStats(invoices: Invoice[]): DashboardStats {
    return useMemo(() => {
        const now = new Date()
        const currentMonth = now.getMonth()
        const currentYear = now.getFullYear()

        let entwuerfe = 0
        let offeneRechnungen = 0
        let ueberfaellig = 0
        let bezahltDiesenMonat = 0

        for (const invoice of invoices) {
            if (invoice.status === 'draft') {
                entwuerfe++
            } else if (invoice.status === 'issued') {
                const dueDate = new Date(invoice.paymentDueAt)
                if (dueDate < now) {
                    ueberfaellig++
                } else {
                    offeneRechnungen++
                }
            } else if (invoice.status === 'paid') { // noch kein paidAt-Feld im Backend, issuedAt wird verwendet
                const paidDate = new Date(invoice.issuedAt)
                if (paidDate.getMonth() === currentMonth && paidDate.getFullYear() === currentYear) {
                    bezahltDiesenMonat++
                }
            }
        }

        return { entwuerfe, offeneRechnungen, ueberfaellig, bezahltDiesenMonat }
    }, [invoices])
}