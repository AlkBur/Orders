package products

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"Orders/internal/database"
	_ "modernc.org/sqlite"
)

func orgsTable() database.Table {
	return database.Must(database.NewTable("organizations",
		database.String("uuid").NotNull().Unique(),
		database.String("name").NotNull(),
		database.String("api_key").NotNull().Default(""),
		database.Bool("active").NotNull().Default(true),
		database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
		database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
	))
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.OpenPath(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	s := database.NewSchema()
	if err := s.Register(orgsTable()); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(Table); err != nil {
		t.Fatal(err)
	}
	if err := s.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	return db
}

func insertOrg(t *testing.T, db *sql.DB, id, name string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO organizations (uuid, name, api_key, active, created_at, updated_at)
		VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, name, "key_"+id); err != nil {
		t.Fatalf("insert org %s: %v", id, err)
	}
}

func TestStore_New(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)

	p := store.New()
	if p.ID != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected nil UUID, got %s", p.ID)
	}
	if !p.Active {
		t.Fatal("expected active by default")
	}
}

func TestStore_SaveInsert(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Test Org")

	p := store.New()
	p.OrganizationID = "org1"
	p.Name = "Test Product"
	p.Unit = "шт"

	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	if p.ID == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("expected generated ID after insert")
	}

	got, err := store.Get(context.Background(), "org1", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Product" || got.Unit != "шт" || !got.Active {
		t.Fatal("unexpected product data")
	}
}

func TestStore_SaveUpdate(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Test Org")

	p := store.New()
	p.OrganizationID = "org1"
	p.Name = "Original"
	p.Unit = "шт"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	p.Name = "Updated"
	p.Unit = "кг"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(context.Background(), "org1", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" || got.Unit != "кг" {
		t.Fatalf("expected 'Updated'/'кг', got '%s'/'%s'", got.Name, got.Unit)
	}
}

func TestStore_SaveInsertWithoutOrg(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)

	p := store.New()
	p.Name = "No Org"
	if err := store.Save(context.Background(), p); err != ErrOrganizationRequired {
		t.Fatalf("expected ErrOrganizationRequired, got %v", err)
	}
}

func TestStore_SaveOrgNotFound(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)

	p := store.New()
	p.OrganizationID = "nonexistent"
	p.Name = "Bad Org"
	if err := store.Save(context.Background(), p); err != ErrOrganizationNotFound {
		t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Test Org")

	if _, err := store.Get(context.Background(), "org1", "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_ListAll(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Org One")
	insertOrg(t, db, "org2", "Org Two")

	p1 := store.New()
	p1.OrganizationID = "org1"
	p1.Name = "Product A"
	store.Save(context.Background(), p1)

	p2 := store.New()
	p2.OrganizationID = "org2"
	p2.Name = "Product B"
	store.Save(context.Background(), p2)

	list, err := store.List(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 products, got %d", len(list))
	}
}

func TestStore_ListByOrg(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Org One")

	p := store.New()
	p.OrganizationID = "org1"
	p.Name = "Only Product"
	store.Save(context.Background(), p)

	list, err := store.List(context.Background(), "org1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 product, got %d", len(list))
	}
}

func TestStore_Delete(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Test Org")

	p := store.New()
	p.OrganizationID = "org1"
	p.Name = "To Delete"
	store.Save(context.Background(), p)

	if err := store.Delete(context.Background(), "org1", p.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(context.Background(), "org1", p.ID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestStore_DeleteNotFound(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)

	if err := store.Delete(context.Background(), "org1", "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_SynchronizeInsert(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Org")

	items := []Product{
		{ID: "ext-1", Name: "From 1C", Unit: "шт", Active: true},
	}

	result, err := store.Synchronize(context.Background(), "org1", items)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	got, err := store.Get(context.Background(), "org1", "ext-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "From 1C" || got.Unit != "шт" {
		t.Fatalf("unexpected product data: %+v", got)
	}
}

func TestStore_SynchronizeUpdate(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Org")

	store.Synchronize(context.Background(), "org1", []Product{
		{ID: "ext-1", Name: "Original", Unit: "шт", Active: true},
	})

	result, err := store.Synchronize(context.Background(), "org1", []Product{
		{ID: "ext-1", Name: "Updated", Unit: "кг", Active: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", result.Updated)
	}

	got, _ := store.Get(context.Background(), "org1", "ext-1")
	if got.Name != "Updated" || got.Unit != "кг" || got.Active {
		t.Fatalf("unexpected product data: %+v", got)
	}
}

func TestStore_SynchronizeDuplicateInRequest(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Org")

	_, err := store.Synchronize(context.Background(), "org1", []Product{
		{ID: "dup", Name: "A", Unit: "шт", Active: true},
		{ID: "dup", Name: "B", Unit: "шт", Active: true},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestStore_SynchronizeRejectsEmptyID(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Org")

	_, err := store.Synchronize(context.Background(), "org1", []Product{
		{ID: "", Name: "Bad", Unit: "шт", Active: true},
	})
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestStore_GetOrganizationName(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "ООО Ромашка")

	p := store.New()
	p.OrganizationID = "org1"
	p.Name = "Test"
	p.Unit = "шт"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(context.Background(), "org1", p.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.OrganizationName != "ООО Ромашка" {
		t.Fatalf("expected 'ООО Ромашка', got %q", got.OrganizationName)
	}
}

func TestStore_ListOrganizationName(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "ООО Ромашка")

	p := store.New()
	p.OrganizationID = "org1"
	p.Name = "Test"
	p.Unit = "шт"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(context.Background(), "org1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 product, got %d", len(list))
	}

	if list[0].OrganizationName != "ООО Ромашка" {
		t.Fatalf("expected 'ООО Ромашка', got %q", list[0].OrganizationName)
	}
}

func TestStore_SynchronizeDifferentOrgs(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1", "Org1")
	insertOrg(t, db, "org2", "Org2")

	store.Synchronize(context.Background(), "org1", []Product{
		{ID: "shared", Name: "In Org1", Unit: "шт", Active: true},
	})
	store.Synchronize(context.Background(), "org2", []Product{
		{ID: "shared", Name: "In Org2", Unit: "шт", Active: true},
	})

	list, err := store.List(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 total, got %d", len(list))
	}
}
