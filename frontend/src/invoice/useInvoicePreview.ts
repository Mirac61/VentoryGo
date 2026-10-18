import { useEffect, useState } from 'react'
import { apiBlob } from '../auth/api'

// Refetches the rendered PDF whenever `revision` changes (a PATCH succeeded).
// The previous PDF stays in `pdfBytes` until the new one has arrived, so the
// preview never flashes or goes blank. A failed fetch keeps the old PDF too.
export function useInvoicePreview(invoiceId: string, revision: number) {
    const [pdfBytes, setPdfBytes] = useState<ArrayBuffer | null>(null)
    const [error, setError] = useState<Error | null>(null)

    useEffect(() => {
        const controller = new AbortController()

        apiBlob(`/api/invoices/${invoiceId}/pdf`, { signal: controller.signal })
            .then(async (blob) => {
                const bytes = await blob.arrayBuffer()
                if (controller.signal.aborted) return
                setPdfBytes(bytes)
                setError(null)
            })
            .catch((caught: unknown) => {
                if (controller.signal.aborted) return
                setError(caught instanceof Error ? caught : new Error(String(caught)))
            })

        return () => controller.abort()
    }, [invoiceId, revision])

    return { pdfBytes, error }
}