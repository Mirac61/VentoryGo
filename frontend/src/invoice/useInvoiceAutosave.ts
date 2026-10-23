import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../auth/api'
import type { Invoice, Issuer, LineItem } from './Invoice'

// Mirrors backend InvoicePatch.
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

export type PatchField = keyof InvoicePatchBody

// Shown to the user when the backend rejects a field.
export const PATCH_FIELD_LABEL: Record<PatchField, string> = {
    title: 'Name der Rechnung',
    paymentDueAt: 'Fälligkeitsdatum',
    serviceDate: 'Leistungsdatum',
    currency: 'Währung',
    sender: 'Absender',
    recipient: 'Empfänger',
    notes: 'Anmerkung',
    vatExempt: 'Umsatzsteuer-Befreiung',
    items: 'Positionen',
}

export type SaveState = 'idle' | 'saving' | 'saved' | 'error' | 'frozen'

interface Options {
    delayMs?: number
    // Called once when the server answers 409: the invoice was issued meanwhile.
    onFrozen: () => void
}

function hasStatus(caught: unknown, status: number): boolean {
    return caught instanceof ApiError && caught.status === status
}

function toError(caught: unknown): Error {
    return caught instanceof Error ? caught : new Error(String(caught))
}

export function useInvoiceAutosave(invoiceId: string, { delayMs = 500, onFrozen }: Options) {
    const [state, setState] = useState<SaveState>('idle')
    const [error, setError] = useState<Error | null>(null)
    // Fields the backend refused and that have not been saved successfully since.
    const [rejected, setRejected] = useState<PatchField[]>([])
    const [lastSaved, setLastSaved] = useState<Invoice | null>(null)
    // Counts successful PATCHes. The preview refetches whenever it changes.
    const [revision, setRevision] = useState(0)

    const pending = useRef<InvoicePatchBody>({})
    const rejectedFields = useRef(new Set<PatchField>())
    const timer = useRef<number | undefined>(undefined)
    const running = useRef<Promise<boolean> | null>(null)
    const frozen = useRef(false)
    const onFrozenRef = useRef(onFrozen)

    useEffect(() => {
        onFrozenRef.current = onFrozen
    }, [onFrozen])

    const patchOnce = useCallback(
        async (body: InvoicePatchBody) => {
            const invoice = await api<Invoice>(`/api/invoices/${invoiceId}`, { method: 'PATCH', body })
            setLastSaved(invoice)
            setRevision((value) => value + 1)
            for (const field of Object.keys(body) as PatchField[]) {
                rejectedFields.current.delete(field)
            }
        },
        [invoiceId],
    )

    // PATCHes run strictly one after another, so responses cannot arrive out of
    // order. Edits made while a request is in flight are merged and sent next.
    const drain = useCallback(async (): Promise<boolean> => {
        // Errors that retrying field by field cannot fix.
        function fail(caught: unknown, body: InvoicePatchBody): false {
            if (hasStatus(caught, 409)) {
                frozen.current = true
                pending.current = {}
                setState('frozen')
                onFrozenRef.current()
                return false
            }
            // Network or server trouble: keep the edit, the next save retries it.
            // A newer edit of the same field wins.
            pending.current = { ...body, ...pending.current }
            setError(toError(caught))
            setState('error')
            return false
        }

        while (Object.keys(pending.current).length > 0) {
            const body = pending.current
            pending.current = {}
            setState('saving')

            try {
                await patchOnce(body)
            } catch (caught) {
                if (!hasStatus(caught, 422)) return fail(caught, body)

                // The backend rejects the whole patch when a single field is
                // invalid. Retry field by field: valid fields still get saved and
                // only the rejected ones are dropped and reported.
                const fields = Object.keys(body) as PatchField[]
                if (fields.length === 1) {
                    rejectedFields.current.add(fields[0])
                    continue
                }
                for (const field of fields) {
                    try {
                        await patchOnce({ [field]: body[field] } as InvoicePatchBody)
                    } catch (fieldCaught) {
                        if (!hasStatus(fieldCaught, 422)) return fail(fieldCaught, body)
                        rejectedFields.current.add(field)
                    }
                }
            }
        }

        setRejected([...rejectedFields.current])
        if (rejectedFields.current.size > 0) {
            setError(new Error('The backend rejected some fields.'))
            setState('error')
            return false
        }
        setError(null)
        setState('saved')
        return true
    }, [patchOnce])

    const flushNow = useCallback((): Promise<boolean> => {
        window.clearTimeout(timer.current)
        if (frozen.current) return Promise.resolve(false)
        running.current ??= drain().finally(() => {
            running.current = null
        })
        return running.current
    }, [drain])

    // Leaving the editor (e.g. back to the dashboard) must not drop an edit that
    // is still waiting for its debounce. The request outlives the component.
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

    return { save, flushNow, state, error, rejected, lastSaved, revision }
}