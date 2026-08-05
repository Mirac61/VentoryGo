package invoice

import (
	"errors"
	"net/http"
	"time"

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
	v.RegisterValidation("notzero", func(fl validator.FieldLevel) bool {
		t, ok := fl.Field().Interface().(time.Time)
		if !ok {
			return false
		}
		return !t.IsZero()
	})
	v.RegisterValidation("iban", func(fl validator.FieldLevel) bool {
		return validateIban(fl.Field().String())
	})
	v.RegisterValidation("currency", func(fl validator.FieldLevel) bool {
		return validateCurrency(fl.Field().String())
	})
}

func ownerFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing user in context"})
		return "", false
	}
	userID, ok := value.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user in context"})
		return "", false
	}
	return userID.String(), true
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(c *gin.Context) {
	ownerID, ok := ownerFromContext(c)
	if !ok {
		return
	}
	var invoice Invoice
	if err := c.ShouldBindJSON(&invoice); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.service.Create(invoice, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) GetByID(c *gin.Context) {
	ownerID, ok := ownerFromContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	invoice, err := h.service.GetByID(id, ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, invoice)
}

func (h *Handler) GetAll(c *gin.Context) {
	ownerID, ok := ownerFromContext(c)
	if !ok {
		return
	}
	invoices, err := h.service.GetAll(ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, invoices)
}

func (h *Handler) Delete(c *gin.Context) {
	ownerID, ok := ownerFromContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	err := h.service.Delete(id, ownerID)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrNotDeletable) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Update(c *gin.Context) {
	ownerID, ok := ownerFromContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var input Invoice

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.service.Update(id, input, ownerID)

	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrNotUpdatable) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrInvalidInput) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) PartialUpdate(c *gin.Context) {
	ownerID, ok := ownerFromContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var input InvoicePatch

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.service.PartialUpdate(id, input, ownerID)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrInvalidInput) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrNotUpdatable) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) Issue(c *gin.Context) {
	ownerID, ok := ownerFromContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	issued, err := h.service.Issue(id, ownerID)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrInvalidTransition) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if mfe, ok := errors.AsType[*MissingFieldsError](err); ok {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":         err.Error(),
			"missingFields": mfe.Fields,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, issued)
}
