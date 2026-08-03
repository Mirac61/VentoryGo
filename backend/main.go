package main

import (
	"context"
	"log"
	"os"

	"github.com/Mirac61/VentoryGo/backend/internal/auth"
	"github.com/Mirac61/VentoryGo/backend/internal/db"
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	connString := os.Getenv("DATABASE_URL")
	pool, err := db.NewPool(ctx, connString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("connected to database successfully")

	numbering, err := invoice.NumberingFromEnv()
	if err != nil {
		log.Fatalf("failed to read invoice numbering config: %v", err)
	}

	cookieSecure, err := auth.CookieSecureFromEnv()
	if err != nil {
		log.Fatalf("failed to read cookie config: %v", err)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	repo := invoice.NewPostgresRepository(pool)
	service := invoice.NewServiceWithNumbering(repo, numbering)
	handler := invoice.NewHandler(service)
	authRepo := auth.NewPostgresRepository(pool)
	authService := auth.NewService(authRepo, auth.NewPostgresSessionStore(pool))
	authHandler := auth.NewHandler(authService, cookieSecure)

	r.POST("/api/invoices", handler.Create)
	r.POST("/api/invoices/:id/issue", handler.Issue)
	r.GET("/api/invoices", handler.GetAll)
	r.GET("/api/invoices/:id", handler.GetByID)
	r.DELETE("/api/invoices/:id", handler.Delete)
	r.PUT("/api/invoices/:id", handler.Update)
	r.PATCH("/api/invoices/:id", handler.PartialUpdate)
	r.POST("/api/auth/register", authHandler.Register)
	r.POST("/api/auth/login", authHandler.Login)

	r.Run(":8080")
}
