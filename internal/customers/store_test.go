package customers

import (
	"context"
	"database/sql"
	"testing"

	"Orders/internal/common"
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

func insertOrg(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO organizations (uuid, name, api_key) VALUES (?, ?, ?)`,
		id, "Test Org", "test_key_"+id)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
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
	if !common.IsNilUUID(c.ID) {
		t.Fatal("expected nil ID by default")
	}
}

func TestSave_Create(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	c := store.New()
	c.OrganizationID = "org1"
	c.Name = "Test Customer"

	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if common.IsNilUUID(c.ID) {
		t.Fatal("expected generated ID")
	}

	got, err := store.Get(context.Background(), "org1", c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Customer" {
		t.Fatalf("expected 'Test Customer', got '%s'", got.Name)
	}
	if !got.Active {
		t.Fatal("expected active")
	}
}

func TestSave_Update(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	c := store.New()
	c.OrganizationID = "org1"
	c.Name = "Original"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	c.Name = "Updated"
	c.Active = false
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(context.Background(), "org1", c.ID)
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
	c.ID = common.NilUUID
	c.OrganizationID = common.NilUUID
	c.Name = "No Org"
	if err := store.Save(context.Background(), c); err != ErrOrganizationRequired {
		t.Fatalf("expected ErrOrganizationRequired, got %v", err)
	}
}

func TestSave_OrganizationNotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	c := store.New()
	c.OrganizationID = "nonexistent"
	c.Name = "No Org"
	if err := store.Save(context.Background(), c); err != ErrOrganizationNotFound {
		t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestSave_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	c := &Customer{
		OrganizationID: "org1",
		ID:             "nonexistent-id",
		Name:           "Nope",
	}
	if err := store.Save(context.Background(), c); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSave_NilUUIDNotStored(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	c := store.New()
	c.OrganizationID = "org1"
	c.Name = "Never Nil"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if common.IsNilUUID(c.ID) {
		t.Fatal("NilUUID should not exist after Save")
	}

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM customers WHERE id = '00000000-0000-0000-0000-000000000000'`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count > 0 {
		t.Fatal("NilUUID found in database — this must never happen")
	}
}

func TestSynchronize_NilUUIDNotStored(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	_, err := store.Synchronize(context.Background(), "org1", []Customer{
		{ID: "ext-1", Name: "From 1C", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM customers WHERE id = '00000000-0000-0000-0000-000000000000'`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count > 0 {
		t.Fatal("NilUUID found in database after Synchronize")
	}
}

func TestSave_OrganizationImmutable(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")
	insertOrg(t, db, "org2")

	c := store.New()
	c.OrganizationID = "org1"
	c.Name = "Original"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	// Try to save with different org_id — UPDATE won't find the record
	c.OrganizationID = "org2"
	if err := store.Save(context.Background(), c); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound when org_id changed, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	c := store.New()
	c.OrganizationID = "org1"
	c.Name = "To Delete"
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(context.Background(), "org1", c.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(context.Background(), "org1", c.ID); err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	if err := store.Delete(context.Background(), "org1", "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList_ByOrganization(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")
	insertOrg(t, db, "org2")

	c1 := store.New()
	c1.OrganizationID = "org1"
	c1.Name = "A"
	if err := store.Save(context.Background(), c1); err != nil {
		t.Fatal(err)
	}

	c2 := store.New()
	c2.OrganizationID = "org1"
	c2.Name = "B"
	if err := store.Save(context.Background(), c2); err != nil {
		t.Fatal(err)
	}

	_ = store.New()
	c3 := store.New()
	c3.OrganizationID = "org2"
	c3.Name = "C"
	if err := store.Save(context.Background(), c3); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(context.Background(), "org1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(list))
	}
	if list[0].OrganizationID != "org1" || list[1].OrganizationID != "org1" {
		t.Fatal("expected only org1 customers")
	}
}

func TestList_AllOrganizations(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")
	insertOrg(t, db, "org2")

	c1 := store.New()
	c1.OrganizationID = "org1"
	c1.Name = "A"
	if err := store.Save(context.Background(), c1); err != nil {
		t.Fatal(err)
	}

	c2 := store.New()
	c2.OrganizationID = "org2"
	c2.Name = "B"
	if err := store.Save(context.Background(), c2); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(context.Background(), common.NilUUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(list))
	}
}

func TestList_Empty(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	list, err := store.List(context.Background(), "org1")
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
	insertOrg(t, db, "org1")

	items := []Customer{
		{ID: "ext-1", Name: "From 1C", Active: true},
		{ID: "ext-2", Name: "Also from 1C", Active: true},
	}

	result, err := store.Synchronize(context.Background(), "org1", items)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 2 {
		t.Fatalf("expected 2 inserted, got %d", result.Inserted)
	}
	if result.Updated != 0 {
		t.Fatalf("expected 0 updated, got %d", result.Updated)
	}

	got, err := store.Get(context.Background(), "org1", "ext-1")
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
	insertOrg(t, db, "org1")

	_, err := store.Synchronize(context.Background(), "org1", []Customer{
		{ID: "ext-1", Name: "Original", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.Synchronize(context.Background(), "org1", []Customer{
		{ID: "ext-1", Name: "Updated", Active: false},
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

	got, err := store.Get(context.Background(), "org1", "ext-1")
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
	insertOrg(t, db, "org1")

	_, err := store.Synchronize(context.Background(), "org1", []Customer{
		{ID: "ext-1", Name: "A", Active: true},
		{ID: "ext-1", Name: "B", Active: true},
	})
	if err == nil {
		t.Fatal("expected error for duplicate in request")
	}
}

func TestList_OrderedByName(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)
	insertOrg(t, db, "org1")

	for _, name := range []string{"C", "A", "B"} {
		c := store.New()
		c.OrganizationID = "org1"
		c.Name = name
		if err := store.Save(context.Background(), c); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.List(context.Background(), "org1")
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
	insertOrg(t, db, "org1")
	insertOrg(t, db, "org2")

	// Sync to org1
	result, err := store.Synchronize(context.Background(), "org1", []Customer{
		{ID: "shared-id", Name: "In Org1", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	// Sync same external ID to org2 (should be separate record)
	result, err = store.Synchronize(context.Background(), "org2", []Customer{
		{ID: "shared-id", Name: "In Org2", Active: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	// Verify both exist independently
	list, err := store.List(context.Background(), common.NilUUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 total, got %d", len(list))
	}
}
