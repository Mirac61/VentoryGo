import { api } from '../auth/api'
import type { Invoice } from './Invoice'

const DEFAULT_PAYMENT_TERM_DAYS = 14

// Creates an empty draft. The backend only insists on a due date and one valid
// line item, everything else is filled in by the editor via PATCH. The seed
// row is replaced by the first real row the user enters.
//
// vatExempt (§ 19 UStG) is deliberately not set here: it belongs to the
// company profile, not to a single invoice, and the editor never toggles it.
// Until the backend sets it from the profile, every draft starts with 19 %.
export function createDraft(): Promise<Invoice> {
    const dueDate = new Date()
    dueDate.setDate(dueDate.getDate() + DEFAULT_PAYMENT_TERM_DAYS)

    return api<Invoice>('/api/invoices', {
        method: 'POST',
        body: {
            paymentDueAt: dueDate.toISOString(),
            sender: { country: 'Deutschland' },
            recipient: { country: 'Deutschland' },
            items: [{ description: 'Leistung', quantity: 1, unitPrice: 0, unit: '', vatRate: 1900 }],
        },
    })
}