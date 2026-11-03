import { useEffect, useState } from 'react'
import { api } from '../auth/api'
import type { Invoice } from './Invoice'


interface UseInvoicesResult {
    invoices: Invoice[]
    // Erster Ladevorgang: es gibt noch keine Daten, die angezeigt werden koennten.
    // Nur dann zeigen Dashboard und Kacheln Platzhalter.
    fetching: boolean
    // Nachladen, waehrend die alten Daten schon auf dem Schirm stehen. Die
    // Anzeige bleibt dabei stehen, statt bei jedem Klick auf "Aktualisieren"
    // kurz auf Platzhalter zurueckzuspringen.
    refreshing: boolean
    fetchResult: Error | null
    refetchInvoices: () => void
}

export function useInvoices(): UseInvoicesResult {
    const [invoices, setInvoices] = useState<Invoice[]>([])
    const [fetching, setFetching] = useState(true)
    const [refreshing, setRefreshing] = useState(false)
    const [fetchResult, setFetchResult] = useState<Error | null>(null)
    const [refetchFlag, setRefetchFlag] = useState(0)

    useEffect(() => {
        let cancelled = false

        setRefreshing(true)

        async function loadInvoices() {
            try {
                const response = await api<Invoice[]>('/api/invoices', { method: 'GET' })
                if (!cancelled) {
                    setInvoices(response)
                    setFetchResult(null)
                }
            } catch (error) {
                if (!cancelled) {
                    setFetchResult(new Error(error instanceof Error ? error.message : String(error)))
                }
            } finally {
                if (!cancelled) {
                    setFetching(false)
                    setRefreshing(false)
                }
            }
        }

        loadInvoices()

        return () => { cancelled = true }
    }, [refetchFlag])

    function refetchInvoices() {
        setRefetchFlag(f => f + 1)
    }

    return { invoices, fetching, refreshing, fetchResult, refetchInvoices }
}
