package database

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func assertTableExists(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
		name,
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("table %q does not exist", name)
	}
}

func assertVersion(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	v, err := getVersion(db)
	if err != nil {
		t.Fatal(err)
	}
	if v != expected {
		t.Fatalf("expected version %d, got %d", expected, v)
	}
}

func register(t *testing.T, s *Schema, table Table) {
	t.Helper()
	if err := s.Register(table); err != nil {
		t.Fatal(err)
	}
}

func addMigration(t *testing.T, s *Schema, m Migration) {
	t.Helper()
	if err := s.AddMigration(m); err != nil {
		t.Fatal(err)
	}
}

func TestRegister_DuplicateTableName(t *testing.T) {
	s := NewSchema()
	register(t, s, Must(NewTable("items", Int("id").PrimaryKey())))
	err := s.Register(Must(NewTable("items", Int("id").PrimaryKey())))
	if err == nil {
		t.Fatal("expected error for duplicate table name")
	}
}

func TestAddMigration_VersionTooLow(t *testing.T) {
	s := NewSchema()
	err := s.AddMigration(Migration{Version: 1, Name: "too low"})
	if err == nil {
		t.Fatal("expected error for version <= InitialSchemaVersion")
	}
}

func TestAddMigration_DuplicateVersion(t *testing.T) {
	s := NewSchema()
	addMigration(t, s, Migration{Version: 2, Name: "first"})
	err := s.AddMigration(Migration{Version: 2, Name: "second"})
	if err == nil {
		t.Fatal("expected error for duplicate version")
	}
}

func TestAddMigration_Gap(t *testing.T) {
	s := NewSchema()
	addMigration(t, s, Migration{Version: 2, Name: "v2"})
	err := s.AddMigration(Migration{Version: 4, Name: "skip v3"})
	if err == nil {
		t.Fatal("expected error for gap")
	}
}

func TestAddMigration_Valid(t *testing.T) {
	s := NewSchema()
	addMigration(t, s, Migration{Version: 2, Name: "v2"})
	addMigration(t, s, Migration{Version: 3, Name: "v3"})
	addMigration(t, s, Migration{Version: 4, Name: "v4"})
}

func TestRunMigrations_FreshDB(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("items",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name").NotNull(),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	assertTableExists(t, db, "items")
	assertVersion(t, db, 1)
}

func TestRunMigrations_FreshDB_MultipleTables(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("a",
		Int("id").PrimaryKey(),
		String("name"),
	)))
	register(t, schema, Must(NewTable("b",
		String("uuid").PrimaryKey(),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	assertTableExists(t, db, "a")
	assertTableExists(t, db, "b")
	assertVersion(t, db, 1)
}

func TestRunMigrations_UpToDate(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("items",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name"),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	// Second run should be a no-op
	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	assertVersion(t, db, 1)
}

func TestRunMigrations_PendingMigration(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("items",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name"),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	addMigration(t, schema, Migration{
		Version: 2,
		Name:    "add column x",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "ALTER TABLE items ADD COLUMN x INTEGER")
			return err
		},
	})

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	assertVersion(t, db, 2)

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('items') WHERE name='x'`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("column x was not added")
	}
}

func TestRunMigrations_MultiplePending(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("items",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name"),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	addMigration(t, schema, Migration{
		Version: 2,
		Name:    "add x",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "ALTER TABLE items ADD COLUMN x INTEGER")
			return err
		},
	})

	addMigration(t, schema, Migration{
		Version: 3,
		Name:    "add y",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "ALTER TABLE items ADD COLUMN y INTEGER")
			return err
		},
	})

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	assertVersion(t, db, 3)
}

func TestRunMigrations_VersionAhead(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("items",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name"),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	if err := setVersion(db, 5); err != nil {
		t.Fatal(err)
	}

	err := schema.RunMigrations(db)
	if err == nil {
		t.Fatal("expected error for version ahead")
	}
}

func TestRunMigrations_GapDetectedBeforeRun(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("items",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name"),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	err := schema.AddMigration(Migration{
		Version: 3,
		Name:    "skip version 2",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected error for gap in AddMigration")
	}
}

func TestRunMigrations_DuplicateDetectedBeforeRun(t *testing.T) {
	db := openDB(t)

	schema := NewSchema()
	register(t, schema, Must(NewTable("items",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name"),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	addMigration(t, schema, Migration{
		Version: 2,
		Name:    "first v2",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			return nil
		},
	})

	err := schema.AddMigration(Migration{
		Version: 2,
		Name:    "second v2",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected error for duplicate version in AddMigration")
	}
}

func TestRunMigrations_OldFormatDB(t *testing.T) {
	db := openDB(t)

	if err := ensureSystemInfoTable(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT OR REPLACE INTO system_info(key, value)
		VALUES('schema_version', '4')
	`); err != nil {
		t.Fatal(err)
	}

	schema := NewSchema()
	register(t, schema, Must(NewTable("users",
		Int("id").PrimaryKey().AutoIncrement(),
		String("login").NotNull().Unique(),
		String("password_hash"),
	)))

	if err := schema.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	assertVersion(t, db, 1)
}

func TestRunMigrations_OldFormatUnknownVersion(t *testing.T) {
	db := openDB(t)

	if err := ensureSystemInfoTable(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT OR REPLACE INTO system_info(key, value)
		VALUES('schema_version', '99')
	`); err != nil {
		t.Fatal(err)
	}

	schema := NewSchema()
	register(t, schema, Must(NewTable("users",
		Int("id").PrimaryKey(),
	)))

	err := schema.RunMigrations(db)
	if err == nil {
		t.Fatal("expected error for unknown old version")
	}
}

func TestCurrentVersion_Empty(t *testing.T) {
	v := CurrentVersion(nil)
	if v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}

	v = CurrentVersion([]Migration{})
	if v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
}

func TestCurrentVersion_WithMigrations(t *testing.T) {
	v := CurrentVersion([]Migration{
		{Version: 2},
		{Version: 3},
	})
	if v != 3 {
		t.Fatalf("expected 3, got %d", v)
	}
}
