import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../auth/api'
import type { Invoice, Issuer, LineItem } from './Invoice'

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

export type PatchLineItem = Pick<LineItem, 'id' | 'description' | 'quantity' | 'unitPrice' | 'unit' | 'vatRate'>

export type SaveState = 'idle' | 'saving' | 'saved' | 'error' | 'frozen'

interface Options {
    delayMs?: number
    onFrozen: () => void
}

export function useInvoiceAutosave(invoiceId: string, { delayMs = 500, onFrozen }: Options) {
    const [state, setState] = useState<SaveState>('idle')
    const [error, setError] = useState<Error | null>(null)
    const [lastSaved, setLastSaved] = useState<Invoice | null>(null)
    const [revision, setRevision] = useState(0)

    const pending = useRef<InvoicePatchBody>({})
    const timer = useRef<number | undefined>(undefined)
    const running = useRef<Promise<boolean> | null>(null)
    const frozen = useRef(false)
    const onFrozenRef = useRef(onFrozen)

    useEffect(() => {
        onFrozenRef.current = onFrozen
    }, [onFrozen])

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
              if (!(caught instanceof ApiError && caught.status == 422)) {
                  pending.current = { ...body, ...pending.current }
                }
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

    
    useEffect(
        () => () => {
            if (Object.keys(pending.current).length > 0) void flushNow()
        },
        [flushNow],
    )

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