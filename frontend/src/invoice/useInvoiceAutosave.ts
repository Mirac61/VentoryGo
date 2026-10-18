import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../auth/api'
import type { Invoice, Issuer, LineItem } from './Invoice'

// Mirrors backend InvoicePatch. Only these fields can be patched today.
export interface InvoicePatchBody {
    title?: string
    paymentDueAt?: string
    serviceDate?: string
    currency?: string
    sender?: Issuer
    recipient?: Invoice['recipient']
    notes?: string
    vatExempt?: boolean
    items?: PatchLineItem[]
}

export type PatchLineItem = Pick<LineItem, 'description' | 'quantity' | 'unitPrice' | 'unit' | 'vatRate'>

export type SaveState = 'idle' | 'saving' | 'saved' | 'error' | 'frozen'

interface Options {
    delayMs?: number
    // Called once when the server answers 409: the invoice was issued meanwhile.
    onFrozen: () => void
}

export function useInvoiceAutosave(invoiceId: string, { delayMs = 500, onFrozen }: Options) {
    const [state, setState] = useState<SaveState>('idle')
    const [error, setError] = useState<Error | null>(null)
    const [lastSaved, setLastSaved] = useState<Invoice | null>(null)
    // Counts successful PATCHes. The preview refetches whenever it changes.
    const [revision, setRevision] = useState(0)

    const pending = useRef<InvoicePatchBody>({})
    const timer = useRef<number | undefined>(undefined)
    const running = useRef<Promise<boolean> | null>(null)
    const frozen = useRef(false)
    const onFrozenRef = useRef(onFrozen)

    useEffect(() => {
        onFrozenRef.current = onFrozen
    }, [onFrozen])

    useEffect(() => () => window.clearTimeout(timer.current), [])

    // PATCHes run strictly one after another, so responses cannot arrive out of
    // order. Edits made while a request is in flight are merged and sent next.
    const drain = useCallback(async (): Promise<boolean> => {
        while (Object.keys(pending.current).length > 0) {
            const body = pending.current
            pending.current = {}
            setState('saving')
            try {
                const invoice = await api<Invoice>(`/api/invoices/${invoiceId}`, { method: 'PATCH', body })
                setLastSaved(invoice)
                setRevision((value) => value + 1)
            } catch (caught) {
                if (caught instanceof ApiError && caught.status === 409) {
                    frozen.current = true
                    pending.current = {}
                    setState('frozen')
                    onFrozenRef.current()
                    return false
                }
                // Keep the failed edit; a newer edit of the same field wins.
                pending.current = { ...body, ...pending.current }
                setError(caught instanceof Error ? caught : new Error(String(caught)))
                setState('error')
                return false
            }
        }
        setError(null)
        setState('saved')
        return true
    }, [invoiceId])

    const flushNow = useCallback((): Promise<boolean> => {
        window.clearTimeout(timer.current)
        if (frozen.current) return Promise.resolve(false)
        running.current ??= drain().finally(() => {
            running.current = null
        })
        return running.current
    }, [drain])

    const save = useCallback(
        (patch: InvoicePatchBody) => {
            if (frozen.current) return
            pending.current = { ...pending.current, ...patch }
            window.clearTimeout(timer.current)
            timer.current = window.setTimeout(() => void flushNow(), delayMs)
        },
        [delayMs, flushNow],
    )

    return { save, flushNow, state, error, lastSaved, revision }
}