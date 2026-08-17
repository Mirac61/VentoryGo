package invoice

const legalNoticeVatExempt = "Gemäß § 19 UStG wird keine Umsatzsteuer berechnet."

func legalNotices(invoice Invoice) []string {
	if invoice.VatExempt {
		return []string{legalNoticeVatExempt}
	}
	return []string{}
}
