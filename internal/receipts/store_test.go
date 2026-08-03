package receipts

import (
	"context"
	"strconv"
	"testing"
	"time"

	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/testutil"
	"Orders/internal/users"
)

func testDB(t *testing.T) *database.Schema {
	t.Helper()
	schema := database.NewSchema()
	if err := schema.Register(users.Table); err != nil {
		t.Fatal(err)
	}
	if err := schema.Register(organizations.Table); err != nil {
		t.Fatal(err)
	}
	if err := schema.Register(customers.Table); err != nil {
		t.Fatal(err)
	}
	if err := schema.Register(products.Table); err != nil {
		t.Fatal(err)
	}
	if err := schema.Register(Table); err != nil {
		t.Fatal(err)
	}
	if err := schema.Register(ItemsTable); err != nil {
		t.Fatal(err)
	}
	return schema
}

func setupTestData(t *testing.T) (context.Context, *Store, int64, int64) {
	t.Helper()
	ctx := context.Background()
	schema := testDB(t)
	db := testutil.NewTestDB(t, schema)

	userStore := users.NewStore(db)
	if err := userStore.Create(&users.User{UUID: "rec-test-user", Login: "operator", IsAdmin: false}); err != nil {
		t.Fatal(err)
	}

	orgStore := organizations.NewStore(db)
	org := orgStore.New()
	org.UUID = "rec-test-org"
	org.Name = "Test Org"
	org.APIKey = "rec-test-key"
	if err := orgStore.Save(ctx, org); err != nil {
		t.Fatal(err)
	}

	custStore := customers.NewStore(db)
	cust := custStore.New()
	cust.UUID = "rec-test-cust"
	cust.Name = "Test Customer"
	cust.OrganizationID = org.ID
	if err := custStore.Save(ctx, cust); err != nil {
		t.Fatal(err)
	}

	store := NewStore(db)
	return ctx, store, org.ID, 1
}

func TestStore_CreateAndGetByID(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	now := time.Now()
	rec := &Receipt{
		Number:         "000001",
		Date:           now,
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          100.50,
		Status:         "",
		StatusColor:    "",
	}

	items := []ReceiptItem{
		{LineNum: 1, ProductID: 1, Unit: "шт", Quantity: 2, Price: 50.25, Amount: 100.50},
	}

	if err := store.Save(ctx, &Document{Receipt: rec, Items: items}); err != nil {
		t.Fatal(err)
	}

	if rec.ID == 0 {
		t.Fatal("expected ID to be assigned")
	}
	if rec.ExchangeID == "" {
		t.Fatal("expected ExchangeID to be assigned")
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}

	if doc.Receipt.Number != "000001" {
		t.Fatalf("expected Number 000001, got %s", doc.Receipt.Number)
	}
	if doc.Receipt.Total != 100.50 {
		t.Fatalf("expected Total 100.50, got %f", doc.Receipt.Total)
	}
	if doc.Receipt.SentAt != nil {
		t.Fatal("expected SentAt to be nil")
	}
	if len(doc.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(doc.Items))
	}
	if doc.Items[0].LineNum != 1 {
		t.Fatalf("expected LineNum 1, got %d", doc.Items[0].LineNum)
	}
}

func TestStore_GeneratesNumber(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	first := &Receipt{OrganizationID: orgID, Date: time.Now()}
	if first.Number != "" {
		t.Fatal("expected number to remain empty before save")
	}
	if err := store.Save(ctx, &Document{Receipt: first}); err != nil {
		t.Fatal(err)
	}
	if first.Number != "000001" {
		t.Fatalf("expected generated number 000001, got %s", first.Number)
	}

	second := &Receipt{OrganizationID: orgID, Date: time.Now()}
	if err := store.Save(ctx, &Document{Receipt: second}); err != nil {
		t.Fatal(err)
	}
	if second.Number != "000002" {
		t.Fatalf("expected generated number 000002, got %s", second.Number)
	}
}

func TestStore_CreateAndUpdate(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := &Receipt{
		Number:         "000002",
		Date:           time.Now(),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          200,
		Status:         "",
		StatusColor:    "",
	}

	items := []ReceiptItem{
		{LineNum: 1, ProductID: 1, Unit: "шт", Quantity: 1, Price: 200, Amount: 200},
	}

	if err := store.Save(ctx, &Document{Receipt: rec, Items: items}); err != nil {
		t.Fatal(err)
	}

	rec.Total = 250
	newItems := []ReceiptItem{
		{LineNum: 1, ProductID: 1, Unit: "шт", Quantity: 1, Price: 250, Amount: 250},
	}

	if err := store.Save(ctx, &Document{Receipt: rec, Items: newItems}); err != nil {
		t.Fatal(err)
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.Total != 250 {
		t.Fatalf("expected Total 250 after update, got %f", doc.Receipt.Total)
	}
}

func TestStore_List(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	for i := 1; i <= 3; i++ {
		rec := &Receipt{
			Number:         "00000" + itoa(i),
			Date:           time.Now(),
			OrganizationID: orgID,
			UserID:         1,
			CustomerID:     1,
			Total:          float64(i * 100),
			Status:         "",
			StatusColor:    "",
		}
		if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.List(ctx, ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 receipts, got %d", len(list))
	}
}

func TestStore_DeleteByID(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := &Receipt{
		Number:         "000010",
		Date:           time.Now(),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          500,
		Status:         "",
		StatusColor:    "",
	}

	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteByID(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}

	_, err := store.GetByID(ctx, rec.ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_Synchronize_UpdateStatus(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := &Receipt{
		Number:         "000020",
		Date:           time.Now(),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          300,
		Status:         "",
		StatusColor:    "",
	}

	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	status := "Обработан"
	color := "success"
	uuid := "1c-uuid-001"

	err := store.Synchronize(ctx, []ReceiptUpdate{
		{
			ExchangeID:  rec.ExchangeID,
			UUID:        &uuid,
			Status:      &status,
			StatusColor: &color,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}

	if doc.Receipt.UUID != "1c-uuid-001" {
		t.Fatalf("expected UUID 1c-uuid-001, got %s", doc.Receipt.UUID)
	}
	if doc.Receipt.Status != "Обработан" {
		t.Fatalf("expected Status Обработан, got %s", doc.Receipt.Status)
	}
	if doc.Receipt.StatusColor != "success" {
		t.Fatalf("expected StatusColor success, got %s", doc.Receipt.StatusColor)
	}
}

func TestStore_Synchronize_PartialUpdate(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := &Receipt{
		Number:         "000021",
		Date:           time.Now(),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          300,
		Status:         "",
		StatusColor:    "",
	}

	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	status := "Отменен"
	err := store.Synchronize(ctx, []ReceiptUpdate{
		{
			ExchangeID: rec.ExchangeID,
			Status:     &status,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.Status != "Отменен" {
		t.Fatalf("expected Status Отменен, got %s", doc.Receipt.Status)
	}
	if doc.Receipt.StatusColor != "" {
		t.Fatalf("expected StatusColor to remain empty, got %s", doc.Receipt.StatusColor)
	}
}

func TestStore_Synchronize_UUIDReassignment(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := &Receipt{
		Number:         "000022",
		Date:           time.Now(),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          300,
		Status:         "",
		StatusColor:    "",
	}

	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	uuid1 := "uuid-first"
	uuid2 := "uuid-second"

	err := store.Synchronize(ctx, []ReceiptUpdate{
		{ExchangeID: rec.ExchangeID, UUID: &uuid1},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.Synchronize(ctx, []ReceiptUpdate{
		{ExchangeID: rec.ExchangeID, UUID: &uuid2},
	})
	if err != ErrUUIDAlreadyAssigned {
		t.Fatalf("expected ErrUUIDAlreadyAssigned, got %v", err)
	}
}

func TestStore_Synchronize_ExchangeIDNotFound(t *testing.T) {
	ctx, store, _, _ := setupTestData(t)

	status := "test"
	err := store.Synchronize(ctx, []ReceiptUpdate{
		{ExchangeID: "nonexistent", Status: &status},
	})
	if err != ErrExchangeIDNotFound {
		t.Fatalf("expected ErrExchangeIDNotFound, got %v", err)
	}
}

func TestStore_SentAt(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	now := time.Now()
	rec := &Receipt{
		Number:         "000030",
		Date:           now,
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          100,
		SentAt:         &now,
		Status:         "",
		StatusColor:    "",
	}

	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.SentAt == nil {
		t.Fatal("expected SentAt to be non-nil")
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
