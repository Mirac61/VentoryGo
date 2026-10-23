import { useEffect, useState } from 'react'
import { apiBlob } from '../auth/api'

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