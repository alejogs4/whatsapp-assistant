package sqlite_test

import (
	"context"
	"testing"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/storage/sqlite"
)

func openMigratedTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() failed: %v", err)
	}
	return db
}

func TestOpenAndMigrate_CreatesAllTablesAndEnablesForeignKeys(t *testing.T) {
	db := openMigratedTestDB(t)

	var fkEnabled int
	if err := db.SQL().QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled); err != nil {
		t.Fatalf("query PRAGMA foreign_keys failed: %v", err)
	}
	if fkEnabled != 1 {
		t.Fatalf("expected PRAGMA foreign_keys=1, got %d", fkEnabled)
	}

	expectedTables := []string{
		"users",
		"learning_paths",
		"mastery_ledger",
		"touchpoint_sessions",
		"learning_path_chunks",
	}
	for _, table := range expectedTables {
		var name string
		err := db.SQL().QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %q to exist, got err: %v", table, err)
		}
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	db := openMigratedTestDB(t)
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("second Migrate() failed: %v", err)
	}
}
