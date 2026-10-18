import { useEffect, useState } from 'react'
import { api } from '../auth/api'
import type { Invoice } from './Invoice'

// Loads one invoice. While neither `invoice` nor `error` is set, it is loading.
export function useInvoice(id: string) {
    const [invoice, setInvoice] = useState<Invoice | null>(null)
    const [error, setError] = useState<Error | null>(null)

    useEffect(() => {
        let cancelled = false

        setInvoice(null)
        setError(null)

        api<Invoice>(`/api/invoices/${id}`, { method: 'GET' })
            .then((loaded) => {
                if (!cancelled) setInvoice(loaded)
            })
            .catch((caught: unknown) => {
                if (!cancelled) setError(caught instanceof Error ? caught : new Error(String(caught)))
            })

        return () => {
            cancelled = true
        }
    }, [id])

    return { invoice, error }
}
