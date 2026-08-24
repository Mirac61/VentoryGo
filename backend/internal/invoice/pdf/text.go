package pdf

import (
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"

	"fmt"
	"strconv"
)

func infoBlockRows(inv invoice.Invoice) [][2]string {
	const dateLayout = "02.01.2006"

	number := "Entwurf"
	if inv.InvoiceNumber != nil {
		number = *inv.InvoiceNumber
	}
	rows := [][2]string{
		{"Rechnungsnr.", number},
		{"Rechnungsdatum", inv.IssuedAt.Format(dateLayout)},
		{"Leistungsdatum", inv.ServiceDate.Format(dateLayout)},
		{"Zahlbar bis", inv.PaymentDueAt.Format(dateLayout)},
	}
	if inv.Sender.VatID != "" {
		rows = append(rows, [2]string{"USt-IdNr.", inv.Sender.VatID})
	} else if inv.Sender.TaxNumber != "" {
		rows = append(rows, [2]string{"Steuernummer", inv.Sender.TaxNumber})
	}
	return rows
}

func serviceSentence(inv invoice.Invoice) string {
	return fmt.Sprintf("Für die erbrachten Leistungen vom %s berechnen wir Ihnen wie folgt:",
		inv.ServiceDate.Format("02.01.2006"))
}

// §14 UStG: breakdown per tax rate.
func vatRowsOf(inv invoice.Invoice) [][2]string {
	if len(inv.VatBreakdown) == 0 {
		return [][2]string{{"zzgl. Umsatzsteuer", formatMoney(inv.VATAmount, inv.Currency)}}
	}
	rows := make([][2]string, 0, len(inv.VatBreakdown))
	for _, entry := range inv.VatBreakdown {
		rows = append(rows, [2]string{
			fmt.Sprintf("zzgl. %s USt auf %s", formatVatRate(entry.VatRate), formatAmount(entry.NetAmount)),
			formatMoney(entry.VatAmount, inv.Currency),
		})
	}
	return rows
}

func senderOneLiner(sender invoice.Issuer) string {
	return fmt.Sprintf("%s · %s %s · %s", sender.Street, sender.Zip, sender.City, sender.Country)
}

// Contact line; the address is already in the return-address line.
func senderContactLine(sender invoice.Issuer) string {
	parts := []string{}
	if sender.Email != "" {
		parts = append(parts, sender.Email)
	}
	if sender.Phone != "" {
		parts = append(parts, sender.Phone)
	}
	if len(parts) == 0 {
		return senderOneLiner(sender)
	}
	return joinWithBullet(parts)
}

func joinWithBullet(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return out
}

func recipientLines(recipient invoice.Contact) []string {
	return []string{
		recipient.Street,
		fmt.Sprintf("%s %s", recipient.Zip, recipient.City),
		recipient.Country,
	}
}

// ASCII-only uppercase to avoid changing umlauts.
func upper(s string) string {
	out := []byte(s)
	for i := 0; i < len(out); i++ {
		if out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 32
		}
	}
	return string(out)
}

// VatRate is in basis points (1900 = 19%).
func formatVatRate(basisPoints int) string {
	if basisPoints%100 == 0 {
		return fmt.Sprintf("%d %%", basisPoints/100)
	}
	return fmt.Sprintf("%d,%d %%", basisPoints/100, (basisPoints%100)/10)
}

func formatQuantityWithUnit(q invoice.Quantity, unit string) string {
	if unit == "" {
		return formatQuantity(q)
	}
	return formatQuantity(q) + " " + unit
}

// MarshalJSON trims trailing zeros.
func formatQuantity(q invoice.Quantity) string {
	b, _ := q.MarshalJSON()
	return string(b)
}

func formatMoney(amount invoice.Money, currency string) string {
	if currency == "" {
		return formatAmount(amount) + " "
	}
	return formatAmount(amount) + " " + currency
}

// Like formatMoney without the currency.
func formatAmount(amount invoice.Money) string {
	cents := int64(amount)
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	whole, frac := cents/100, cents%100
	return fmt.Sprintf("%s%s,%02d", sign, groupThousands(whole), frac)
}

func groupThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}
