package invoice

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Mirac61/VentoryGo/backend/internal/httperror"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	register := func(tag string, fn validator.Func) {
		if err := v.RegisterValidation(tag, fn); err != nil {
			panic(fmt.Sprintf("invoice: RegisterValidation %q fehlgeschlagen: %v", tag, err))
		}
	}
	register("notzero", func(fl validator.FieldLevel) bool {
		t, ok := fl.Field().Interface().(time.Time)
		if !ok {
			return false
		}
		return !t.IsZero()
	})
	register("iban", func(fl validator.FieldLevel) bool {
		return validateIban(fl.Field().String())
	})
	register("currency", func(fl validator.FieldLevel) bool {
		return validateCurrency(fl.Field().String())
	})
}

func ownerFromContext(c *gin.Context) (string, error) {
	value, exists := c.Get("user_id")
	if !exists {
		return "", ErrMissingOwner
	}
	userID, ok := value.(uuid.UUID)
	if !ok {
		return "", ErrMissingOwner
	}
	return userID.String(), nil
}

// PDFRenderer is injected because pdf imports this package; calling it
// directly from here would be an import cycle.
type PDFRenderer func(Invoice) ([]byte, error)

type Handler struct {
	service   *Service
	renderPDF PDFRenderer
}

func NewHandler(service *Service, renderPDF PDFRenderer) *Handler {
	return &Handler{
		service:   service,
		renderPDF: renderPDF,
	}
}

func (h *Handler) Create(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	var invoice Invoice
	if !httperror.Bind(c, &invoice) {
		return
	}
	created, err := h.service.Create(invoice, ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) GetByID(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	id := c.Param("id")
	invoice, err := h.service.GetByID(id, ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, invoice)
}

func (h *Handler) GetAll(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	invoices, err := h.service.GetAll(ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, invoices)
}

func (h *Handler) Delete(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	id := c.Param("id")
	err = h.service.Delete(id, ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Update(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	id := c.Param("id")
	var input Invoice

	if !httperror.Bind(c, &input) {
		return
	}
	updated, err := h.service.Update(id, input, ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) PartialUpdate(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	id := c.Param("id")
	var input InvoicePatch

	if !httperror.Bind(c, &input) {
		return
	}
	updated, err := h.service.PartialUpdate(id, input, ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) DownloadPDF(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	if h.renderPDF == nil {
		httperror.WriteError(c, ErrNoPDFRenderer)
		return
	}
	id := c.Param("id")
	inv, err := h.service.GetByID(id, ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	pdfBytes, err := h.renderPDF(inv)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	filename := id
	if inv.InvoiceNumber != nil {
		filename = *inv.InvoiceNumber
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename+".pdf"))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func (h *Handler) Issue(c *gin.Context) {
	ownerID, err := ownerFromContext(c)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	id := c.Param("id")
	issued, err := h.service.Issue(id, ownerID)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, issued)
}
