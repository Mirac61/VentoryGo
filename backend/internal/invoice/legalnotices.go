package invoice

const legalNoticeVatExempt = "Gemäß § 19 UStG wird keine Umsatzsteuer berechnet."

// LegalNotices derives the mandatory legal notices for an invoice, e.g. the
// §19 UStG exemption notice for VAT-exempt invoices.
func LegalNotices(invoice Invoice) []string {
	if invoice.VatExempt {
		return []string{legalNoticeVatExempt}
	}
	return []string{}
}
