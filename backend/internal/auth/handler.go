package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const defaultCookieSecure = true

type Handler struct {
	service      *Service
	cookieSecure bool
}

func NewHandler(service *Service, cookieSecure bool) *Handler {
	return &Handler{
		service:      service,
		cookieSecure: cookieSecure,
	}
}

func CookieSecureFromEnv() (bool, error) {
	value := os.Getenv("COOKIE_SECURE")
	if value == "" {
		return defaultCookieSecure, nil
	}

	secure, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid COOKIE_SECURE %q: %w", value, err)
	}
	return secure, nil
}

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			fields := gin.H{}
			for _, fieldErr := range validationErrs {
				fields[strings.ToLower(fieldErr.Field())] = fieldErr.Tag()
			}
			c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": fields})
			return false
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return false
	}
	return true
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if !bindJSON(c, &req) {
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

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}

	_, token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session", token, int(h.service.sessionTTL.Seconds()), "/", "", h.cookieSecure, true)
	c.Status(http.StatusNoContent)
}
