package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			fields := gin.H{}
			for _, fieldErr := range validationErrs {
				fields[strings.ToLower(fieldErr.Field())] = fieldErr.Tag()
			}
			c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": fields})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	user, err := h.service.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusCreated, user)
}
