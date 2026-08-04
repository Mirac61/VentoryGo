package main

import (
	"time"

	"github.com/Mirac61/VentoryGo/backend/internal/auth"
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"
	"github.com/gin-gonic/gin"
)

func newRouter(
	invoices *invoice.Handler,
	authHandler *auth.Handler,
	sessions auth.SessionStore,
	sessionTTL time.Duration,
	cookieSecure bool,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	grp := r.Group("/api/invoices", auth.RequireAuth(sessions, sessionTTL, cookieSecure))
	grp.GET("", invoices.GetAll)
	grp.GET("/:id", invoices.GetByID)
	grp.POST("", invoices.Create)
	grp.POST("/:id/issue", invoices.Issue)
	grp.DELETE("/:id", invoices.Delete)
	grp.PUT("/:id", invoices.Update)
	grp.PATCH("/:id", invoices.PartialUpdate)
	r.POST("/api/auth/register", authHandler.Register)
	r.POST("/api/auth/login", authHandler.Login)
	r.POST("/api/auth/logout", authHandler.Logout)
	r.GET("/api/auth/me", auth.RequireAuth(sessions, sessionTTL, cookieSecure), authHandler.Me)
	return r
}
