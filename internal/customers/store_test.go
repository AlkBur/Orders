package customers

import (
	"context"
	"database/sql"
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
		t.Fatalf("insert org: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestNew(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	c := store.New()
	if c == nil {
		t.Fatal("expected non-nil customer")
	}
	if !c.Active {
		t.Fatal("expected active by default")
	}
	if c.UUID != "" {
		t.Fatal("expected empty UUID by default")
	}
}

func TestSave_Create(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	c := store.New()
	c.OrganizationID = orgID
	c.Name = "Test Customer"
	c.UUID = "cust-test-create"

	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByExternal(context.Background(), orgID, c.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Customer" {
		t.Fatalf("expected 'Test Customer', got '%s'", got.Name)
	}
	if !got.Active {
		t.Fatal("expected active")
	}
	if got.OrganizationName != "Org1" {
		t.Fatalf("expected 'Org1', got '%s'", got.OrganizationName)
	}
}

func TestSave_Update(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	c := store.New()
	c.OrganizationID = orgID
	c.Name = "Original"
	c.UUID = "cust-test-update-1"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	c.Name = "Updated"
	c.Active = false
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByExternal(context.Background(), orgID, c.UUID)
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

func TestSave_OrganizationRequired(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	c := store.New()
	c.Name = "No Org"
	if err := store.Save(context.Background(), c); err != ErrOrganizationRequired {
		t.Fatalf("expected ErrOrganizationRequired, got %v", err)
	}
}

func TestSave_OrganizationNotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	c := store.New()
	c.OrganizationID = 999
	c.Name = "No Org"
	if err := store.Save(context.Background(), c); err != ErrOrganizationNotFound {
		t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestSave_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	c := &Customer{
		OrganizationID: orgID,
		ID:             999,
		Name:           "Nope",
	}
	if err := store.Save(context.Background(), c); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSave_OrganizationImmutable(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID1 := insertOrg(t, db, "Org1", "key1")
	orgID2 := insertOrg(t, db, "Org2", "key2")

	c := store.New()
	c.OrganizationID = orgID1
	c.Name = "Original"
	c.UUID = "cust-test-immutable-1"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	// changing org_id in struct should NOT affect the stored record
	c.OrganizationID = orgID2
	c.Name = "Updated"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OrganizationID != orgID1 {
		t.Fatal("organization_id should be immutable after creation")
	}
	if got.Name != "Updated" {
		t.Fatalf("expected 'Updated', got '%s'", got.Name)
	}
}

func TestDeleteByExternal(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	c := store.New()
	c.OrganizationID = orgID
	c.Name = "To Delete"
	c.UUID = "cust-test-del-ext"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteByExternal(context.Background(), orgID, c.UUID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetByExternal(context.Background(), orgID, c.UUID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestDeleteByExternal_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if err := store.DeleteByExternal(context.Background(), 1, "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteByID(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	c := store.New()
	c.OrganizationID = orgID
	c.Name = "To Delete"
	c.UUID = "cust-test-del-id"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteByID(context.Background(), c.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetByID(context.Background(), c.ID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestList_ByOrganization(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID1 := insertOrg(t, db, "Org1", "key1")
	orgID2 := insertOrg(t, db, "Org2", "key2")

	c1 := store.New()
	c1.OrganizationID = orgID1
	c1.Name = "A"
	c1.UUID = "cust-list-a"
	if err := store.Save(context.Background(), c1); err != nil {
		t.Fatal(err)
	}

	c2 := store.New()
	c2.OrganizationID = orgID1
	c2.Name = "B"
	c2.UUID = "cust-list-b"
	if err := store.Save(context.Background(), c2); err != nil {
		t.Fatal(err)
	}

	c3 := store.New()
	c3.OrganizationID = orgID2
	c3.Name = "C"
	c3.UUID = "cust-list-c"
	if err := store.Save(context.Background(), c3); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(context.Background(), orgID1)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(list))
	}
	if list[0].OrganizationID != orgID1 || list[1].OrganizationID != orgID1 {
		t.Fatal("expected only org1 customers")
	}
}

func TestList_AllOrganizations(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID1 := insertOrg(t, db, "Org1", "key1")
	orgID2 := insertOrg(t, db, "Org2", "key2")

	c1 := store.New()
	c1.OrganizationID = orgID1
	c1.Name = "A"
	c1.UUID = "cust-list-all-a"
	if err := store.Save(context.Background(), c1); err != nil {
		t.Fatal(err)
	}

	c2 := store.New()
	c2.OrganizationID = orgID2
	c2.Name = "B"
	c2.UUID = "cust-list-all-b"
	if err := store.Save(context.Background(), c2); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(list))
	}
	for _, c := range list {
		if c.OrganizationName != "Org1" && c.OrganizationName != "Org2" {
			t.Fatalf("expected organization name, got '%s' for customer %d", c.OrganizationName, c.ID)
		}
	}
}

func TestList_Empty(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	list, err := store.List(context.Background(), orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 customers, got %d", len(list))
	}
}

func TestSynchronize_Insert(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	items := []Customer{
		{UUID: "ext-1", Name: "From 1C", Active: true},
		{UUID: "ext-2", Name: "Also from 1C", Active: true},
	}

	result, err := store.Synchronize(context.Background(), orgID, items)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 2 {
		t.Fatalf("expected 2 inserted, got %d", result.Inserted)
	}
	if result.Updated != 0 {
		t.Fatalf("expected 0 updated, got %d", result.Updated)
	}

	got, err := store.GetByExternal(context.Background(), orgID, "ext-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "From 1C" {
		t.Fatalf("expected 'From 1C', got '%s'", got.Name)
	}
}

func TestSynchronize_Update(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	_, err := store.Synchronize(context.Background(), orgID, []Customer{
		{UUID: "ext-1", Name: "Original", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.Synchronize(context.Background(), orgID, []Customer{
		{UUID: "ext-1", Name: "Updated", Active: false},
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

	got, err := store.GetByExternal(context.Background(), orgID, "ext-1")
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

func TestSynchronize_DuplicateInRequest(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	_, err := store.Synchronize(context.Background(), orgID, []Customer{
		{UUID: "ext-1", Name: "A", Active: true},
		{UUID: "ext-1", Name: "B", Active: true},
	})
	if err == nil {
		t.Fatal("expected error for duplicate in request")
	}
}

func TestList_OrderedByName(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	for _, name := range []string{"C", "A", "B"} {
		c := store.New()
		c.OrganizationID = orgID
		c.Name = name
		c.UUID = "cust-order-" + name
		if err := store.Save(context.Background(), c); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.List(context.Background(), orgID)
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

func TestSynchronize_WithOrganizationIDParam(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID1 := insertOrg(t, db, "Org1", "key1")
	orgID2 := insertOrg(t, db, "Org2", "key2")

	// Sync to org1
	result, err := store.Synchronize(context.Background(), orgID1, []Customer{
		{UUID: "shared-id", Name: "In Org1", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	// Sync same external ID to org2 (should be separate record)
	result, err = store.Synchronize(context.Background(), orgID2, []Customer{
		{UUID: "shared-id", Name: "In Org2", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	// Verify both exist independently
	list, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 total, got %d", len(list))
	}
}

func TestSave_UUIDPreserved(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	c := store.New()
	c.OrganizationID = orgID
	c.Name = "UUID Test"
	c.UUID = "preserved-uuid"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if c.UUID != "preserved-uuid" {
		t.Fatal("UUID should be preserved after Save")
	}
}

func TestSynchronize_NilUUIDNotStored(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	_, err := store.Synchronize(context.Background(), orgID, []Customer{
		{UUID: "ext-1", Name: "From 1C", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSynchronize_RejectsEmptyUUID(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	orgID := insertOrg(t, db, "Org1", "key1")

	_, err := store.Synchronize(context.Background(), orgID, []Customer{
		{UUID: "", Name: "Bad", Active: true},
	})
	if err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	s := database.NewSchema()
	if err := s.Register(organizations.Table); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(Table); err != nil {
		t.Fatal(err)
	}
	db, err := database.OpenPath(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := s.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	return db
}
