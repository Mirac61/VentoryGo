package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const renewInterval = 15 * time.Minute

func RequireAuth(sessions SessionStore, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		hash := hashToken(token)
		session, err := sessions.Get(c.Request.Context(), hash)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		now := time.Now().UTC()
		lastTouch := session.ExpiresAt.Add(-ttl)
		if lastTouch.Before(now.Add(-renewInterval)) {
			_ = sessions.Touch(c.Request.Context(), hash, now.Add(ttl))
		}
		c.Set("user_id", session.UserID)
		c.Next()
	}
}
