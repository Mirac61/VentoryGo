package auth

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/Mirac61/VentoryGo/backend/internal/httperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func setSessionCookie(c *gin.Context, token string, maxAge int, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session", token, maxAge, "/", "", secure, true)
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if !httperror.Bind(c, &req) {
		return
	}

	user, err := h.service.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !httperror.Bind(c, &req) {
		return
	}

	_, token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	setSessionCookie(c, token, int(h.service.sessionTTL.Seconds()), h.cookieSecure)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Logout(c *gin.Context) {
	token, cookieErr := c.Cookie("session")
	if cookieErr == nil {
		if err := h.service.Logout(c.Request.Context(), token); err != nil {
			setSessionCookie(c, "", -1, h.cookieSecure)
			httperror.WriteError(c, err)
			return
		}
	}
	setSessionCookie(c, "", -1, h.cookieSecure)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	value, _ := c.Get("user_id")
	id, ok := value.(uuid.UUID)
	if !ok {
		httperror.WriteError(c, fmt.Errorf("user id in context is not a uuid"))
		return
	}
	user, err := h.service.Me(c.Request.Context(), id)
	if err != nil {
		httperror.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}
