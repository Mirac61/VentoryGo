package main

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrateDatabase applies pending migrations before the server starts. A failed
// migration aborts the start instead of serving against a stale schema.
func migrateDatabase(pool *pgxpool.Pool) error {
	if value := os.Getenv("MIGRATE_ON_START"); value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid MIGRATE_ON_START %q: %w", value, err)
		}
		if !enabled {
			log.Print("automatic migrations disabled (MIGRATE_ON_START=false)")
			return nil
		}
	}

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("reading embedded migrations: %w", err)
	}

	driver, err := pgxmigrate.WithInstance(stdlib.OpenDBFromPool(pool), &pgxmigrate.Config{})
	if err != nil {
		return fmt.Errorf("creating migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}

	version, _, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		log.Print("no migrations applied")
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	log.Printf("database schema at version %d", version)
	return nil
}
