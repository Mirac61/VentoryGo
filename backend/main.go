package main

import (
	"context"
	"log"
	"os"

	"github.com/Mirac61/VentoryGo/backend/internal/auth"
	"github.com/Mirac61/VentoryGo/backend/internal/db"
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"
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

	cookieSecure, err := auth.CookieSecureFromEnv()
	if err != nil {
		log.Fatalf("failed to read cookie config: %v", err)
	}

	sessionTTL, err := auth.SessionTTLFromEnv()
	if err != nil {
		log.Fatalf("failed to read session config: %v", err)
	}

	repo := invoice.NewPostgresRepository(pool)
	service := invoice.NewService(repo)
	handler := invoice.NewHandler(service)
	authRepo := auth.NewPostgresRepository(pool)
	sessionStore := auth.NewPostgresSessionStore(pool)
	authService := auth.NewServiceWithSessionTTL(authRepo, sessionStore, sessionTTL)
	authHandler := auth.NewHandler(authService, cookieSecure)

	r := newRouter(handler, authHandler, sessionStore, sessionTTL, cookieSecure)

	r.Run(":8080")
}
