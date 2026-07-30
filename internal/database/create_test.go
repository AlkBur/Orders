package database

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

type columnInfo struct {
	cid     int
	name    string
	ctype   string
	notnull bool
	dflt    sql.NullString
	pk      int
}

func execSQL(t *testing.T, db *sql.DB, sql string) {
	t.Helper()
	if _, err := db.Exec(sql); err != nil {
		t.Fatalf("exec: %v\nSQL: %s", err, sql)
	}
}

func getColumns(t *testing.T, db *sql.DB, table string) []columnInfo {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer rows.Close()

	var cols []columnInfo
	for rows.Next() {
		var c columnInfo
		if err := rows.Scan(&c.cid, &c.name, &c.ctype, &c.notnull, &c.dflt, &c.pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		cols = append(cols, c)
	}
	return cols
}

func TestCreateSQL_Users(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	execSQL(t, db, usersTable().CreateSQL())

	cols := getColumns(t, db, "users")
	if len(cols) != 6 {
		t.Fatalf("expected 6 columns, got %d", len(cols))
	}

	assertColumn(t, cols, 0, "id", "INTEGER", false, "", 1)
	assertColumn(t, cols, 1, "login", "TEXT", true, "", 0)
	assertColumn(t, cols, 2, "password_hash", "TEXT", true, "''", 0)
	assertColumn(t, cols, 3, "is_admin", "INTEGER", true, "0", 0)
	assertColumn(t, cols, 4, "created_at", "DATETIME", true, "CURRENT_TIMESTAMP", 0)
	assertColumn(t, cols, 5, "updated_at", "DATETIME", true, "CURRENT_TIMESTAMP", 0)
}

func TestCreateSQL_Sessions(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	execSQL(t, db, sessionsTable().CreateSQL())

	cols := getColumns(t, db, "sessions")
	if len(cols) != 9 {
		t.Fatalf("expected 9 columns, got %d", len(cols))
	}

	assertColumn(t, cols, 0, "id", "TEXT", false, "", 1)
	assertColumn(t, cols, 1, "user_id", "INTEGER", false, "", 0)
}

func TestCreateSQL_Customers(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	execSQL(t, db, customersTable().CreateSQL())

	cols := getColumns(t, db, "customers")
	if len(cols) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(cols))
	}

	assertColumn(t, cols, 0, "uuid", "TEXT", false, "", 1)
	assertColumn(t, cols, 1, "name", "TEXT", true, "", 0)
	assertColumn(t, cols, 2, "active", "INTEGER", true, "1", 0)
}

func TestCreateSQL_Organizations(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	execSQL(t, db, organizationsTable().CreateSQL())

	cols := getColumns(t, db, "organizations")
	if len(cols) != 6 {
		t.Fatalf("expected 6 columns, got %d", len(cols))
	}

	assertColumn(t, cols, 0, "uuid", "TEXT", false, "", 1)
	assertColumn(t, cols, 1, "name", "TEXT", true, "", 0)
	assertColumn(t, cols, 2, "api_key", "TEXT", true, "", 0)
	assertColumn(t, cols, 3, "active", "INTEGER", true, "1", 0)
}

func TestCreateSQLIfNotExists_Idempotent(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sql := organizationsTable().CreateSQLIfNotExists()

	execSQL(t, db, sql)
	execSQL(t, db, sql)
}

func assertColumn(t *testing.T, cols []columnInfo, idx int, name, ctype string, notnull bool, dflt string, pk int) {
	t.Helper()
	c := cols[idx]
	if c.name != name {
		t.Errorf("col[%d] name: expected %q, got %q", idx, name, c.name)
	}
	if c.ctype != ctype {
		t.Errorf("col[%d] %q type: expected %q, got %q", idx, name, ctype, c.ctype)
	}
	if c.notnull != notnull {
		t.Errorf("col[%d] %q notnull: expected %v, got %v", idx, name, notnull, c.notnull)
	}
	if c.dflt.Valid && c.dflt.String != dflt {
		t.Errorf("col[%d] %q default: expected %q, got %q", idx, name, dflt, c.dflt.String)
	}
	if !c.dflt.Valid && dflt != "" {
		t.Errorf("col[%d] %q default: expected %q, got NULL", idx, name, dflt)
	}
	if c.pk != pk {
		t.Errorf("col[%d] %q pk: expected %d, got %d", idx, name, pk, c.pk)
	}
}

func usersTable() Table {
	return Must(NewTable("users",
		Int("id").PrimaryKey().AutoIncrement(),
		String("login").NotNull().Unique(),
		String("password_hash").NotNull().Default(""),
		Bool("is_admin").NotNull().Default(false),
		DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
		DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
	))
}

func sessionsTable() Table {
	return Must(NewTable("sessions",
		String("id").PrimaryKey(),
		Int("user_id").References("users", "id").OnDelete("SET NULL"),
		String("flash_type").NotNull().Default(""),
		String("flash_message").NotNull().Default(""),
		String("values_json").NotNull().Default("{}"),
		String("user_agent").NotNull().Default(""),
		DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
		DateTime("last_seen_at").NotNull().Default("CURRENT_TIMESTAMP"),
		DateTime("expires_at").NotNull(),
	))
}

func customersTable() Table {
	return Must(NewTable("customers",
		String("uuid").PrimaryKey(),
		String("name").NotNull(),
		Bool("active").NotNull().Default(true),
		DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
		DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
	))
}

func organizationsTable() Table {
	return Must(NewTable("organizations",
		String("uuid").PrimaryKey(),
		String("name").NotNull(),
		String("api_key").NotNull().Unique(),
		Bool("active").NotNull().Default(true),
		DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
		DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
	))
}
