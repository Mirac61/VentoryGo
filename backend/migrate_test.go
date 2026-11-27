package main

import (
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestMigrateDatabaseDisabled(t *testing.T) {
	t.Setenv("MIGRATE_ON_START", "false")

	if err := migrateDatabase(nil); err != nil {
		t.Fatalf("expected no error when disabled, got %v", err)
	}
}

func TestMigrateDatabaseRejectsInvalidToggle(t *testing.T) {
	t.Setenv("MIGRATE_ON_START", "sometimes")

	if err := migrateDatabase(nil); err == nil {
		t.Fatal("expected error for invalid MIGRATE_ON_START")
	}
}

func TestMigrationsEmbedded(t *testing.T) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("embedded migrations: %v", err)
	}
	if _, err := src.First(); err != nil {
		t.Fatalf("first migration: %v", err)
	}
}
