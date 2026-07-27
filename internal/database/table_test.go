package database

import (
	"testing"
)

func TestNewTable_EmptyName(t *testing.T) {
	_, err := NewTable("")
	if err == nil {
		t.Fatal("expected error for empty table name")
	}
}

func TestNewTable_EmptyColumnName(t *testing.T) {
	_, err := NewTable("t", Column{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty column name")
	}
}

func TestNewTable_DuplicateColumn(t *testing.T) {
	_, err := NewTable("t",
		String("a"),
		String("a"),
	)
	if err == nil {
		t.Fatal("expected error for duplicate column")
	}
}

func TestNewTable_TwoPrimaryKeys(t *testing.T) {
	_, err := NewTable("t",
		Int("a").PrimaryKey(),
		Int("b").PrimaryKey(),
	)
	if err == nil {
		t.Fatal("expected error for multiple primary keys")
	}
}

func TestNewTable_AutoIncWithoutPK(t *testing.T) {
	_, err := NewTable("t",
		Int("id").AutoIncrement(),
	)
	if err == nil {
		t.Fatal("expected error for AUTOINCREMENT without PK")
	}
}

func TestNewTable_AutoIncOnString(t *testing.T) {
	_, err := NewTable("t",
		String("id").PrimaryKey().AutoIncrement(),
	)
	if err == nil {
		t.Fatal("expected error for AUTOINCREMENT on non-INTEGER")
	}
}

func TestNewTable_Valid(t *testing.T) {
	_, err := NewTable("users",
		Int("id").PrimaryKey().AutoIncrement(),
		String("name").NotNull(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMust_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	Must(NewTable(""))
}

func TestMust_ReturnsTable(t *testing.T) {
	table := Must(NewTable("t", Int("id").PrimaryKey()))
	if table.Name != "t" {
		t.Fatalf("expected name t, got %s", table.Name)
	}
}
