package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	"Orders/internal/database"

	_ "modernc.org/sqlite"
)

func NewTestDB(t *testing.T, schema *database.Schema) *sql.DB {
	t.Helper()

	db, err := database.OpenPath(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if err := schema.RunMigrations(db); err != nil {
		db.Close()
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
