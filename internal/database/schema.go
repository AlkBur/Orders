// Package database provides the schema builder and migration engine.
//
// Architecture:
//   - Table/Column types describe the target database schema.
//   - CreateSQL() generates CREATE TABLE statements from a Table.
//   - Schema collects tables and migrations, runs them against a database.
//   - Migration type represents a versioned schema change for existing DBs.
//   - New databases are created directly from Table descriptions (no migration replay).
//   - Existing databases evolve through sequential migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
)

const InitialSchemaVersion = 1

type Schema struct {
	tables     []Table
	migrations []Migration
}

func NewSchema() *Schema {
	return &Schema{}
}

func (s *Schema) Register(table Table) error {
	for _, t := range s.tables {
		if t.Name == table.Name {
			return fmt.Errorf("table %q already registered", table.Name)
		}
	}
	s.tables = append(s.tables, table)
	return nil
}

func (s *Schema) AddMigration(m Migration) error {
	if m.Version <= InitialSchemaVersion {
		return fmt.Errorf(
			"migration version must be > %d, got %d",
			InitialSchemaVersion, m.Version,
		)
	}
	for _, existing := range s.migrations {
		if existing.Version == m.Version {
			return fmt.Errorf("duplicate migration version %d", m.Version)
		}
	}
	expected := InitialSchemaVersion + 1
	if len(s.migrations) > 0 {
		expected = s.migrations[len(s.migrations)-1].Version + 1
	}
	if m.Version != expected {
		return fmt.Errorf(
			"migration version must be %d (next), got %d",
			expected, m.Version,
		)
	}
	s.migrations = append(s.migrations, m)
	return nil
}

func (s *Schema) RunMigrations(db *sql.DB) error {
	if err := ensureSystemInfoTable(db); err != nil {
		return fmt.Errorf("system_info: %w", err)
	}

	v, err := getVersion(db)
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}

	codeVersion := CurrentVersion(s.migrations)

	// Fresh database — create all tables from descriptions
	if v == 0 {
		for _, t := range s.tables {
			if _, err := db.Exec(t.CreateSQL()); err != nil {
				return fmt.Errorf("create %q: %w", t.Name, err)
			}
		}
		return setVersion(db, codeVersion)
	}

	// Old-format database (pre-migration system)
	if knownOldVersions[v] {
		return setVersion(db, codeVersion)
	}

	// Already up to date
	if v == codeVersion {
		return nil
	}

	// Database is ahead of code — should not happen
	if v > codeVersion {
		return fmt.Errorf("database version %d > code version %d", v, codeVersion)
	}

	// Apply pending migrations
	for _, m := range s.migrations {
		if m.Version > v && m.Version <= codeVersion {
			if err := applyMigration(db, m); err != nil {
				return fmt.Errorf("migration %d: %w", m.Version, err)
			}
		}
	}

	return nil
}

func applyMigration(db *sql.DB, m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := m.Up(context.Background(), tx); err != nil {
		return fmt.Errorf("migration %d (%s): %w", m.Version, m.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return setVersion(db, m.Version)
}
