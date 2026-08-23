export interface Contact{
  name: string
  street: string
  zip: string
  city: string
  country: string
  email: string
  phone: string
  taxId: string
}

export interface Issuer extends Contact {
  vatId: string
  taxNumber: string
  iban: string
  bic: string
  bankName: string
}

  export interface LineItem {
    id: string
    invoiceId: string
    position: number
    description: string
    quantity: number 
    unitPrice: number
    unit: string
    total: number
    vatRate: number
  }

export interface VATBreakdownEntry{
    vatRate: number
    netAmount: number
    vatAmount: number
}

export interface Invoice {
    id: string
    invoiceNumber: string | null
    status: 'draft' | 'issued' | 'paid' | 'cancelled'
    createdAt: string
    issuedAt: string
    serviceDate: string
    currency: string
    paymentDueAt: string
    sender: Issuer
    recipient: Contact
    items: LineItem[]
    vatBreakdown: VATBreakdownEntry[]
    netTotal: number
    vatAmount: number
    grossTotal: number
    notes: string
}