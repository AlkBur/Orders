package customers

import (
	"context"
	"database/sql"
	"testing"

	"Orders/internal/database"
	"Orders/internal/testutil"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	schema := database.NewSchema()
	if err := schema.Register(Table); err != nil {
		t.Fatalf("register table: %v", err)
	}
	return testutil.NewTestDB(t, schema)
}

func TestSynchronize_EmptyList_EmptyDB(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	result, err := store.Synchronize(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.Inserted != 0 {
		t.Fatalf("expected 0 inserted, got %d", result.Inserted)
	}
	if result.Updated != 0 {
		t.Fatalf("expected 0 updated, got %d", result.Updated)
	}
	if result.Deactivated != 0 {
		t.Fatalf("expected 0 deactivated, got %d", result.Deactivated)
	}
}

func TestSynchronize_FirstSync(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	items := []CustomerSnapshot{
		{UUID: "a", Name: "A"},
		{UUID: "b", Name: "B"},
	}

	result, err := store.Synchronize(context.Background(), items)
	if err != nil {
		t.Fatal(err)
	}

	if result.Inserted != 2 {
		t.Fatalf("expected 2 inserted, got %d", result.Inserted)
	}
	if result.Updated != 0 {
		t.Fatalf("expected 0 updated, got %d", result.Updated)
	}
	if result.Deactivated != 0 {
		t.Fatalf("expected 0 deactivated, got %d", result.Deactivated)
	}
}

func TestSynchronize_Idempotent(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	items := []CustomerSnapshot{
		{UUID: "a", Name: "A"},
		{UUID: "b", Name: "B"},
	}

	if _, err := store.Synchronize(context.Background(), items); err != nil {
		t.Fatal(err)
	}

	result, err := store.Synchronize(context.Background(), items)
	if err != nil {
		t.Fatal(err)
	}

	if result.Inserted != 0 {
		t.Fatalf("expected 0 inserted, got %d", result.Inserted)
	}
	if result.Updated != 0 {
		t.Fatalf("expected 0 updated, got %d", result.Updated)
	}
	if result.Deactivated != 0 {
		t.Fatalf("expected 0 deactivated, got %d", result.Deactivated)
	}
}

func TestSynchronize_UpdateName(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if _, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A"},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A-new"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Inserted != 0 {
		t.Fatalf("expected 0 inserted, got %d", result.Inserted)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", result.Updated)
	}
	if result.Deactivated != 0 {
		t.Fatalf("expected 0 deactivated, got %d", result.Deactivated)
	}
}

func TestSynchronize_Deactivate(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if _, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A"},
		{UUID: "b", Name: "B"},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Inserted != 0 {
		t.Fatalf("expected 0 inserted, got %d", result.Inserted)
	}
	if result.Updated != 0 {
		t.Fatalf("expected 0 updated, got %d", result.Updated)
	}
	if result.Deactivated != 1 {
		t.Fatalf("expected 1 deactivated, got %d", result.Deactivated)
	}
}

func TestSynchronize_Reactivate(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if _, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A"},
		{UUID: "b", Name: "B"},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A"},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A"},
		{UUID: "b", Name: "B"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Inserted != 0 {
		t.Fatalf("expected 0 inserted, got %d", result.Inserted)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated (reactivated), got %d", result.Updated)
	}
	if result.Deactivated != 0 {
		t.Fatalf("expected 0 deactivated, got %d", result.Deactivated)
	}
}

func TestSynchronize_Combined(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if _, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A"},
		{UUID: "b", Name: "B"},
		{UUID: "c", Name: "C"},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := store.Synchronize(context.Background(), []CustomerSnapshot{
		{UUID: "a", Name: "A-new"},
		{UUID: "d", Name: "D"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted (d), got %d", result.Inserted)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated (a), got %d", result.Updated)
	}
	if result.Deactivated != 2 {
		t.Fatalf("expected 2 deactivated (b, c), got %d", result.Deactivated)
	}
}
