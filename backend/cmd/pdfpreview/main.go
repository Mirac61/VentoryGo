package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/Mirac61/VentoryGo/backend/internal/invoice"
	"github.com/Mirac61/VentoryGo/backend/internal/invoice/pdf"
)

func main() {
	outDir := outputDir()
	inv := sampleInvoice()

	designs := []struct {
		name   string
		design pdf.Design
	}{
		{"classic", pdf.Classic},
		{"modern", pdf.Modern},
		{"minimal", pdf.Minimal},
	}

	for _, d := range designs {
		write(outDir, "preview-"+d.name+".pdf", d.design, inv)
	}

	for _, brand := range []struct{ name, hex string }{
		{"brand-gruen", "#3F6B54"},
		{"brand-bordeaux", "#7A3B4A"},
		{"brand-anthrazit", "#3A3F47"},
	} {
		branded := inv
		branded.Sender.AccentColor = brand.hex
		write(outDir, "preview-"+brand.name+".pdf", pdf.Modern, branded)
	}

	write(outDir, "preview-long.pdf", pdf.Modern, longInvoice())
	write(outDir, "preview-exempt.pdf", pdf.Modern, exemptInvoice())
	write(outDir, "preview-nologo.pdf", pdf.Classic, noLogoInvoice())
}

// Previews land next to this file, regardless of the working directory.
func outputDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

func write(dir, name string, design pdf.Design, inv invoice.Invoice) {
	pdfBytes, err := pdf.Generate(inv, design)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), pdfBytes, 0o644); err != nil {
		panic(err)
	}
}

func sampleInvoice() invoice.Invoice {
	invoiceNumber := "2026-0042"
	inv := invoice.Invoice{
		InvoiceNumber: &invoiceNumber,
		Currency:      "EUR",
		CreatedAt:     time.Now(),
		IssuedAt:      time.Now(),
		ServiceDate:   time.Now().Add(-48 * time.Hour),
		PaymentDueAt:  time.Now().Add(14 * 24 * time.Hour),
		Sender: invoice.Issuer{
			Contact: invoice.Contact{
				Name:    "Ventory Handels GmbH",
				Street:  "Königstraße 27",
				Zip:     "70173",
				City:    "Stuttgart",
				Country: "Deutschland",
				Email:   "rechnung@ventory.example",
			},
			VatID: "DE123456789",
			IBAN:  "DE89370400440532013000",
		},
		Recipient: invoice.Contact{
			Name:    "Nordlicht Systeme GmbH",
			Street:  "Hafenstraße 118",
			Zip:     "20359",
			City:    "Hamburg",
			Country: "Deutschland",
		},
		Items: []invoice.LineItem{
			{Position: 1, Description: "Konzeption und Beratung", Quantity: 8000, Unit: "Std.", UnitPrice: 12500, Total: 100000, VatRate: 1900},
			{Position: 2, Description: "Implementierung Warenwirtschaft", Quantity: 24000, Unit: "Std.", UnitPrice: 9500, Total: 228000, VatRate: 1900},
			{Position: 3, Description: "Lizenz Ventory Pro (Jahr)", Quantity: 3000, Unit: "Stk.", UnitPrice: 49000, Total: 147000, VatRate: 1900},
			{Position: 4, Description: "Schulung vor Ort", Quantity: 1000, Unit: "Tag", UnitPrice: 89000, Total: 89000, VatRate: 1900},
		},
		VatBreakdown: []invoice.VATBreakdownEntry{
			{VatRate: 1900, NetAmount: 564000, VatAmount: 107160},
		},
		NetTotal:   564000,
		VATAmount:  107160,
		GrossTotal: 671160,
		Notes:      "Vielen Dank für die gute Zusammenarbeit.",
	}
	inv.Sender.BIC = "COBADEFFXXX"
	inv.Sender.BankName = "Commerzbank Stuttgart"
	inv.Sender.Logo = sampleLogoPNG()
	return inv
}

func sampleLogoPNG() []byte {
	const w, h = 420, 140
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	navy := color.NRGBA{R: 26, G: 58, B: 110, A: 255}

	for y := 10; y < 130; y++ {
		for x := 10; x < 130; x++ {
			inNotch := x > 40 && x < 100 && y > 40 && y < 100 && (x-40) < (y-40)
			if !inNotch {
				img.Set(x, y, navy)
			}
		}
	}
	for i, width := range []int{250, 190, 220} {
		top := 26 + i*34
		for y := top; y < top+18; y++ {
			for x := 160; x < 160+width; x++ {
				img.Set(x, y, navy)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func longInvoice() invoice.Invoice {
	inv := sampleInvoice()
	inv.Items = nil
	net := invoice.Money(0)
	for i := 1; i <= 42; i++ {
		total := invoice.Money(1000 * int64(i))
		inv.Items = append(inv.Items, invoice.LineItem{
			Position: i, Description: "Wartungspauschale Modul " + strconv.Itoa(i),
			Quantity: 1000, Unit: "Stk.", UnitPrice: total, Total: total, VatRate: 1900,
		})
		net += total
	}
	vat := invoice.RoundedVAT(net, 1900)
	inv.NetTotal, inv.VATAmount, inv.GrossTotal = net, vat, net+vat
	inv.VatBreakdown = []invoice.VATBreakdownEntry{{VatRate: 1900, NetAmount: net, VatAmount: vat}}
	return inv
}

func exemptInvoice() invoice.Invoice {
	inv := sampleInvoice()
	inv.VatExempt = true
	inv.VatBreakdown = nil
	inv.VATAmount = 0
	inv.GrossTotal = inv.NetTotal
	inv.LegalNotices = []string{"Gemäß § 19 UStG wird keine Umsatzsteuer berechnet."}
	for i := range inv.Items {
		inv.Items[i].VatRate = 0
	}
	return inv
}

func noLogoInvoice() invoice.Invoice {
	inv := sampleInvoice()
	inv.Sender.Logo = nil
	return inv
}
