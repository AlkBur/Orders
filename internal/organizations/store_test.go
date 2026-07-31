package organizations

import (
	"context"
	"database/sql"
	"testing"

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
	if o.UUID != "" {
		t.Fatal("expected empty UUID by default")
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
	o.UUID = "org-test-save-create"

	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByUUID(context.Background(), o.UUID)
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
	o.UUID = "org-test-update-1"
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	o.Name = "Updated"
	o.Active = false
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByUUID(context.Background(), o.UUID)
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
		ID:     999,
		UUID:   "some-uuid",
		Name:   "Nope",
		Active: true,
	}
	if err := store.Save(context.Background(), o); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByUUID_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	_, err := store.GetByUUID(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	_, err := store.GetByID(context.Background(), 999)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteByUUID(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	o := store.New()
	o.Name = "To Delete"
	o.UUID = "org-test-delete-uuid"
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteByUUID(context.Background(), o.UUID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetByUUID(context.Background(), o.UUID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestDeleteByUUID_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if err := store.DeleteByUUID(context.Background(), "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteByID(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	o := store.New()
	o.Name = "To Delete"
	o.UUID = "org-test-delete-id"
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteByID(context.Background(), o.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetByID(context.Background(), o.ID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestList_Empty(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	list, err := store.List(context.Background(), ListOptions{})
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
		o.UUID = "org-" + name
		if err := store.Save(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.List(context.Background(), ListOptions{})
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
	o.UUID = "org-test-apikey"
	if err := store.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	if o.APIKey == "" {
		t.Fatal("expected generated API key")
	}

	got, err := store.GetByUUID(context.Background(), o.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey == "" {
		t.Fatal("expected API key in database")
	}
}
