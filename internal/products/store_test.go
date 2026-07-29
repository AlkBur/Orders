package products

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"Orders/internal/database"
	"Orders/internal/organizations"
	"Orders/internal/testutil"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	s := database.NewSchema()
	if err := s.Register(organizations.Table); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(Table); err != nil {
		t.Fatal(err)
	}
	return testutil.NewTestDB(t, s)
}

func insertOrg(t *testing.T, db *sql.DB, name, apiKey string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO organizations (uuid, name, api_key, active, created_at, updated_at) VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		"uuid-"+name, name, apiKey)
	if err != nil {
		t.Fatalf("insert org %s: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestStore_New(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	p := store.New()
	if p.UUID != "" {
		t.Fatalf("expected empty UUID, got %s", p.UUID)
	}
	if !p.Active {
		t.Fatal("expected active by default")
	}
}

func TestStore_SaveInsert(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "TestOrg", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "Test Product"
	p.Unit = "шт"
	p.UUID = "prod-test-insert"

	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByExternal(context.Background(), orgID, p.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Product" || got.Unit != "шт" || !got.Active {
		t.Fatal("unexpected product data")
	}
}

func TestStore_SaveUpdate(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "TestOrg", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "Original"
	p.Unit = "шт"
	p.UUID = "prod-test-update-1"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	p.Name = "Updated"
	p.Unit = "кг"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByExternal(context.Background(), orgID, p.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" || got.Unit != "кг" {
		t.Fatalf("expected 'Updated'/'кг', got '%s'/'%s'", got.Name, got.Unit)
	}
}

func TestStore_SaveInsertWithoutOrg(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	p := store.New()
	p.Name = "No Org"
	if err := store.Save(context.Background(), p); err != ErrOrganizationRequired {
		t.Fatalf("expected ErrOrganizationRequired, got %v", err)
	}
}

func TestStore_SaveOrgNotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	p := store.New()
	p.OrganizationID = 999
	p.Name = "Bad Org"
	if err := store.Save(context.Background(), p); err != ErrOrganizationNotFound {
		t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestStore_GetByExternalNotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "TestOrg", "key1")

	if _, err := store.GetByExternal(context.Background(), orgID, "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_ListAll(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID1 := insertOrg(t, db, "Org1", "key1")
	orgID2 := insertOrg(t, db, "Org2", "key2")

	p1 := store.New()
	p1.OrganizationID = orgID1
	p1.Name = "Product A"
	p1.UUID = "prod-list-all-a"
	store.Save(context.Background(), p1)

	p2 := store.New()
	p2.OrganizationID = orgID2
	p2.Name = "Product B"
	p2.UUID = "prod-list-all-b"
	store.Save(context.Background(), p2)

	list, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 products, got %d", len(list))
	}
}

func TestStore_ListByOrg(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "Only Product"
	p.UUID = "prod-only"
	store.Save(context.Background(), p)

	list, err := store.List(context.Background(), orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 product, got %d", len(list))
	}
}

func TestStore_Delete(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "TestOrg", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "To Delete"
	p.UUID = "prod-del-ext"
	store.Save(context.Background(), p)

	if err := store.DeleteByExternal(context.Background(), orgID, p.UUID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetByExternal(context.Background(), orgID, p.UUID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestStore_DeleteNotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if err := store.DeleteByExternal(context.Background(), 1, "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_SynchronizeInsert(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org", "key1")

	items := []Product{
		{UUID: "ext-1", Name: "From 1C", Unit: "шт", Active: true},
	}

	result, err := store.Synchronize(context.Background(), orgID, items)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	got, err := store.GetByExternal(context.Background(), orgID, "ext-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "From 1C" || got.Unit != "шт" {
		t.Fatalf("unexpected product data: %+v", got)
	}
}

func TestStore_SynchronizeUpdate(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org", "key1")

	store.Synchronize(context.Background(), orgID, []Product{
		{UUID: "ext-1", Name: "Original", Unit: "шт", Active: true},
	})

	result, err := store.Synchronize(context.Background(), orgID, []Product{
		{UUID: "ext-1", Name: "Updated", Unit: "кг", Active: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", result.Updated)
	}

	got, _ := store.GetByExternal(context.Background(), orgID, "ext-1")
	if got.Name != "Updated" || got.Unit != "кг" || got.Active {
		t.Fatalf("unexpected product data: %+v", got)
	}
}

func TestStore_SynchronizeDuplicateInRequest(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org", "key1")

	_, err := store.Synchronize(context.Background(), orgID, []Product{
		{UUID: "dup", Name: "A", Unit: "шт", Active: true},
		{UUID: "dup", Name: "B", Unit: "шт", Active: true},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestStore_SynchronizeRejectsEmptyUUID(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org", "key1")

	_, err := store.Synchronize(context.Background(), orgID, []Product{
		{UUID: "", Name: "Bad", Unit: "шт", Active: true},
	})
	if err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func TestStore_GetOrganizationName(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "ООО Ромашка", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "Test"
	p.Unit = "шт"
	p.UUID = "prod-orgname"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByExternal(context.Background(), orgID, p.UUID)
	if err != nil {
		t.Fatal(err)
	}

	if got.OrganizationName != "ООО Ромашка" {
		t.Fatalf("expected 'ООО Ромашка', got %q", got.OrganizationName)
	}
}

func TestStore_ListOrganizationName(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "ООО Ромашка", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "Test"
	p.Unit = "шт"
	p.UUID = "prod-list-orgname"
	if err := store.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(context.Background(), orgID)
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
	db := testDB(t)
	store := NewStore(db)
	orgID1 := insertOrg(t, db, "Org1", "key1")
	orgID2 := insertOrg(t, db, "Org2", "key2")

	store.Synchronize(context.Background(), orgID1, []Product{
		{UUID: "shared", Name: "In Org1", Unit: "шт", Active: true},
	})
	store.Synchronize(context.Background(), orgID2, []Product{
		{UUID: "shared", Name: "In Org2", Unit: "шт", Active: true},
	})

	list, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 total, got %d", len(list))
	}
}

func TestStore_GetByID(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "TestOrg", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "Test"
	p.UUID = "prod-getbyid"
	store.Save(context.Background(), p)

	got, err := store.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test" {
		t.Fatalf("expected 'Test', got '%s'", got.Name)
	}
}

func TestStore_DeleteByID(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "TestOrg", "key1")

	p := store.New()
	p.OrganizationID = orgID
	p.Name = "To Delete"
	p.UUID = "prod-del-id"
	store.Save(context.Background(), p)

	if err := store.DeleteByID(context.Background(), p.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetByID(context.Background(), p.ID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}
