package organizations

import (
	"context"
	"database/sql"
	"testing"

	"Orders/internal/common"
	"Orders/internal/database"
	"Orders/internal/testutil"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	s := database.NewSchema()
	if err := s.Register(Table); err != nil {
		t.Fatal(err)
	}
	return testutil.NewTestDB(t, s)
}

func TestNew(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	o := store.New()
	if o == nil {
		t.Fatal("expected non-nil organization")
	}
	if !common.IsNilUUID(o.UUID) {
		t.Fatal("expected nil UUID by default")
	}
	if !o.Active {
		t.Fatal("expected active by default")
	}
}

func TestSave_Create(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	o := store.New()
	o.Name = "Test Org"

	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	if common.IsNilUUID(o.UUID) {
		t.Fatal("expected generated UUID")
	}

	got, err := store.Get(context.Background(), o.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Org" {
		t.Fatalf("expected 'Test Org', got '%s'", got.Name)
	}
	if !got.Active {
		t.Fatal("expected active")
	}
}

func TestSave_Update(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	o := store.New()
	o.Name = "Original"
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	o.Name = "Updated"
	o.Active = false
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(context.Background(), o.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" {
		t.Fatalf("expected 'Updated', got '%s'", got.Name)
	}
	if got.Active {
		t.Fatal("expected inactive")
	}
}

func TestSave_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	o := &Organization{
		UUID:   "nonexistent-uuid",
		Name:   "Nope",
		Active: true,
	}
	if err := store.Save(context.Background(), o); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	_, err := store.Get(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	o := store.New()
	o.Name = "To Delete"
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(context.Background(), o.UUID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(context.Background(), o.UUID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if err := store.Delete(context.Background(), "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList_Empty(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	list, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 organizations, got %d", len(list))
	}
}

func TestList_OrderedByName(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	for _, name := range []string{"C", "A", "B"} {
		o := store.New()
		o.Name = name
		if err := store.Save(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3, got %d", len(list))
	}
	if list[0].Name != "A" || list[1].Name != "B" || list[2].Name != "C" {
		t.Fatalf("expected sorted order, got %+v", list)
	}
}

func TestSave_GenerateAPIKey(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	o := store.New()
	o.Name = "With API Key"
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	if o.APIKey == "" {
		t.Fatal("expected generated API key")
	}

	got, err := store.Get(context.Background(), o.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey == "" {
		t.Fatal("expected API key in database")
	}
}
