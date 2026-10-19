import type { Invoice, LineItem } from './Invoice'
import type { InvoicePatchBody, PatchLineItem } from './useInvoiceAutosave'


export type ItemMode = 'quantity' | 'fixed'


export interface ItemRow {
    id: string
    description: string
    mode: ItemMode
    quantity: string
    unit: string
    unitPrice: string
    vatRate: number
}

export interface DetailsForm {
    title: string
    serviceDate: string // YYYY-MM-DD or ''
    paymentDueDate: string // YYYY-MM-DD or ''
    currency: string
    notes: string
    rows: ItemRow[]
}

export const VAT_RATES = [0, 700, 1900]
const DEFAULT_VAT_RATE = 1900

export const CURRENCIES = [
    { value: 'EUR', label: 'Euro (EUR)' },
    { value: 'USD', label: 'US-Dollar (USD)' },
    { value: 'GBP', label: 'Britisches Pfund (GBP)' },
    { value: 'CHF', label: 'Schweizer Franken (CHF)' },
]

const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/
const MIN_PLAUSIBLE_YEAR = 2000


export function parseCents(input: string): number | null {
    const match = /^(\d{1,9})(?:[.,](\d{1,2}))?$/.exec(input.trim())
    if (!match) return null
    return Number(match[1]) * 100 + Number((match[2] ?? '').padEnd(2, '0'))
}

// "1,5" -> 1.5. At most three decimal places like the backend, and above zero.
export function parseQuantity(input: string): number | null {
    const match = /^(\d{1,9})(?:[.,](\d{1,3}))?$/.exec(input.trim())
    if (!match) return null
    const value = Number(`${match[1]}.${match[2] ?? '0'}`)
    return value > 0 ? value : null
}

function formatCents(cents: number): string {
    return `${Math.floor(cents / 100)},${String(cents % 100).padStart(2, '0')}`
}

function formatQuantity(quantity: number): string {
    return String(quantity).replace('.', ',')
}

export function formatCentsDisplay(cents: number, currency: string): string {
    const sign = cents < 0 ? '-' : ''
    const absolute = Math.abs(cents)
    const whole = String(Math.floor(absolute / 100)).replace(/\B(?=(\d{3})+(?!\d))/g, '.')
    const fraction = String(absolute % 100).padStart(2, '0')
    return `${sign}${whole},${fraction} ${currency === 'EUR' ? '€' : currency}`
}

export function formatVatRate(basisPoints: number): string {
    if (basisPoints % 100 === 0) return `${basisPoints / 100} %`
    return `${String(basisPoints / 100).replace('.', ',')} %`
}

export function dateOnly(iso: string): string {
    return iso.startsWith('0001') ? '' : iso.slice(0, 10)
}

export function formatDateGerman(date: string): string {
    if (!DATE_PATTERN.test(date)) return ''
    const [year, month, day] = date.split('-')
    return `${day}.${month}.${year}`
}

function isPlausibleDate(value: string): boolean {
    return DATE_PATTERN.test(value) && Number(value.slice(0, 4)) >= MIN_PLAUSIBLE_YEAR
}


function defaultVatRate(vatExempt: boolean): number {
    return vatExempt ? 0 : DEFAULT_VAT_RATE
}

export function newRow(vatExempt: boolean): ItemRow {
    return {
        id: crypto.randomUUID(),
        description: '',
        mode: 'quantity',
        quantity: '1',
        unit: '',
        unitPrice: '',
        vatRate: defaultVatRate(vatExempt),
    }
}

function rowFromItem(item: LineItem): ItemRow {
    return {
        id: item.id,
        description: item.description,
        mode: 'quantity',
        quantity: formatQuantity(item.quantity),
        unit: item.unit,
        unitPrice: formatCents(item.unitPrice),
        vatRate: item.vatRate,
    }
}

export function detailsFromInvoice(invoice: Invoice): DetailsForm {
    return {
        title: invoice.title,
        serviceDate: dateOnly(invoice.serviceDate),
        paymentDueDate: dateOnly(invoice.paymentDueAt),
        currency: invoice.currency || 'EUR',
        notes: invoice.notes,
        rows: invoice.items.map(rowFromItem),
    }
}

export interface RowErrors {
    quantity?: string
    unitPrice?: string
}

export function rowErrors(row: ItemRow): RowErrors {
    const errors: RowErrors = {}
    if (row.mode === 'quantity' && parseQuantity(row.quantity) === null) {
        errors.quantity = 'Menge größer 0, max. 3 Nachkommastellen'
    }
    if (row.unitPrice.trim() !== '' && parseCents(row.unitPrice) === null) {
        errors.unitPrice = 'Betrag wie 49,90'
    }
    return errors
}

// A row is saved only when it is complete and valid. An empty price counts as 0.
function toPatchItem(row: ItemRow, vatExempt: boolean): PatchLineItem | null {
    const description = row.description.trim()
    if (description === '') return null

    const isFixed = row.mode === 'fixed'
    const quantity = isFixed ? 1 : parseQuantity(row.quantity)
    if (quantity === null) return null

    const unitPrice = row.unitPrice.trim() === '' ? 0 : parseCents(row.unitPrice)
    if (unitPrice === null) return null

    return {
        id: row.id,
        description,
        quantity,
        unitPrice,
        unit: isFixed ? '' : row.unit.trim(),
        vatRate: vatExempt ? 0 : row.vatRate,
    }
}

export function isRowSaveable(row: ItemRow, vatExempt: boolean): boolean {
    return toPatchItem(row, vatExempt) !== null
}

export function saveableItems(rows: ItemRow[], vatExempt: boolean): PatchLineItem[] {
    const items = rows.flatMap((row) => {
        const item = toPatchItem(row, vatExempt)
        return item ? [item] : []
    })
    if (items.length > 0) return items

    return [
        {
            id: crypto.randomUUID(),
            description: 'Leistung',
            quantity: 1,
            unitPrice: 0,
            unit: '',
            vatRate: defaultVatRate(vatExempt),
        },
    ]
}


export function detailsPatch(changed: Partial<DetailsForm>, next: DetailsForm, vatExempt: boolean): InvoicePatchBody {
    const patch: InvoicePatchBody = {}
    if (changed.title !== undefined) patch.title = changed.title.trim()
    if (changed.notes !== undefined) patch.notes = changed.notes
    if (changed.currency !== undefined) patch.currency = changed.currency
    if (changed.serviceDate !== undefined && isPlausibleDate(changed.serviceDate)) {
        patch.serviceDate = `${changed.serviceDate}T00:00:00Z`
    }
    if (changed.paymentDueDate !== undefined && isPlausibleDate(changed.paymentDueDate)) {
        patch.paymentDueAt = `${changed.paymentDueDate}T00:00:00Z`
    }
    if (changed.rows !== undefined) patch.items = saveableItems(next.rows, vatExempt)
    return patch
}