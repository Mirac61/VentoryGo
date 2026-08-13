package invoice

import "time"

type InvoicePatch struct {
	PaymentDueAt *time.Time  `json:"paymentDueAt" binding:"omitempty,notzero"`
	Recipient    *Contact    `json:"recipient"`
	Items        *[]LineItem `json:"items" binding:"omitempty,min=1,dive"`
	Notes        *string     `json:"notes"`
}

type InvoiceStatus string

const (
	StatusDraft     InvoiceStatus = "draft"
	StatusIssued    InvoiceStatus = "issued"
	StatusPaid      InvoiceStatus = "paid"
	StatusCancelled InvoiceStatus = "cancelled"
)

type Invoice struct {
	ID            string              `json:"id"`
	InvoiceNumber *string             `json:"invoiceNumber"`
	Status        InvoiceStatus       `json:"status"`
	CreatedAt     time.Time           `json:"createdAt"`
	IssuedAt      time.Time           `json:"issuedAt"`
	ServiceDate   time.Time           `json:"serviceDate"`
	Currency      string              `json:"currency" binding:"omitempty,len=3,currency"`
	PaymentDueAt  time.Time           `json:"paymentDueAt" binding:"required"`
	Sender        Issuer              `json:"sender" binding:"required"`
	Recipient     Contact             `json:"recipient" binding:"required"`
	Items         []LineItem          `json:"items" binding:"required,min=1,dive"`
	VatBreakdown  []VATBreakdownEntry `json:"vatBreakdown"`
	NetTotal      Money               `json:"netTotal"`
	VATAmount     Money               `json:"vatAmount"`
	GrossTotal    Money               `json:"grossTotal"`
	Notes         string              `json:"notes"`
	OwnerID       string              `json:"-"`
}

type Contact struct {
	Name    string `json:"name" binding:"required"`
	Street  string `json:"street" binding:"required"`
	Zip     string `json:"zip" binding:"required"`
	City    string `json:"city" binding:"required"`
	Country string `json:"country" binding:"required"`
	Email   string `json:"email" binding:"omitempty,email"`
	Phone   string `json:"phone"`
	TaxID   string `json:"taxId"`
}

type Issuer struct {
	Contact
	VatID     string `json:"vatId"`
	TaxNumber string `json:"taxNumber"`
	IBAN      string `json:"iban" binding:"omitempty,iban"`
	BIC       string `json:"bic"`
	BankName  string `json:"bankName"`
}

type LineItem struct {
	ID          string `json:"id"`
	InvoiceID   string `json:"invoiceId"`
	Position    int    `json:"position"`
	Description string `json:"description" binding:"required"`
	Quantity    int    `json:"quantity" binding:"gt=0"`
	UnitPrice   Money  `json:"unitPrice" binding:"gte=0"`
	Unit        string `json:"unit"`
	Total       Money  `json:"total"`
	VatRate     int    `json:"vatRate" binding:"required"`
}

type VATBreakdownEntry struct {
	VatRate   int   `json:"vatRate"`
	NetAmount Money `json:"netAmount"`
	VatAmount Money `json:"vatAmount"`
}
