package auth

import (
	"errors"
	"log"
	"time"

	"github.com/Mirac61/VentoryGo/backend/internal/httperror"
	"github.com/gin-gonic/gin"
)

const maxRenewInterval = 15 * time.Minute

func RequireAuth(sessions SessionStore, ttl time.Duration, cookieSecure bool) gin.HandlerFunc {
	renewInterval := min(maxRenewInterval, ttl/2)
	return func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			httperror.WriteError(c, httperror.ErrUnauthenticated)
			c.Abort()
			return
		}
		hash := hashToken(token)
		session, err := sessions.Get(c.Request.Context(), hash)
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				httperror.WriteError(c, httperror.ErrUnauthenticated)
			} else {
				httperror.WriteError(c, err)
			}
			c.Abort()
			return
		}
		now := time.Now().UTC()
		lastTouch := session.ExpiresAt.Add(-ttl)
		if lastTouch.Before(now.Add(-renewInterval)) {
			if err := sessions.Touch(c.Request.Context(), hash, now.Add(ttl)); err != nil {
				log.Printf("session touch: %v", err)
			} else {
				setSessionCookie(c, token, int(ttl.Seconds()), cookieSecure)
			}
		}
		c.Set("user_id", session.UserID)
		c.Next()
	}
}
