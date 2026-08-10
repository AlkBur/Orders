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
	}

	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	status := "Обработан"
	uuid := "1c-uuid-001"

	err := store.Synchronize(ctx, []ReceiptUpdate{
		{
			ExchangeID: rec.ExchangeID,
			UUID:       &uuid,
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

	if doc.Receipt.UUID != "1c-uuid-001" {
		t.Fatalf("expected UUID 1c-uuid-001, got %s", doc.Receipt.UUID)
	}
	if doc.Receipt.Status != "Обработан" {
		t.Fatalf("expected Status Обработан, got %s", doc.Receipt.Status)
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

func saveReceipt(t *testing.T, store *Store, ctx context.Context, orgID int64, number string, sent bool) *Receipt {
	t.Helper()
	rec := &Receipt{
		Number:         number,
		Date:           time.Now(),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          100,
		Status:         "",
	}
	now := time.Now()
	if sent {
		rec.SentAt = &now
	}
	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestStore_ListAvailableForSync(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	sent := saveReceipt(t, store, ctx, orgID, "100001", true)
	unsent := saveReceipt(t, store, ctx, orgID, "100002", false)

	docs, err := store.ListAvailableForSync(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 documented, got %d", len(docs))
	}
	if docs[0].Receipt.ID != sent.ID {
		t.Fatalf("expected receipt %d, got %d", sent.ID, docs[0].Receipt.ID)
	}
	if docs[0].Receipt.Number != sent.Number {
		t.Fatalf("expected Number %s, got %s", sent.Number, docs[0].Receipt.Number)
	}
	if docs[0].Receipt.CustomerName != "Test Customer" {
		t.Fatalf("expected CustomerName Test Customer, got %s", docs[0].Receipt.CustomerName)
	}
	if docs[0].Receipt.CustomerUUID != "rec-test-cust" {
		t.Fatalf("expected CustomerUUID rec-test-cust, got %s", docs[0].Receipt.CustomerUUID)
	}
	if len(docs[0].Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(docs[0].Items))
	}
	if unsent.ID == 0 {
		t.Fatal("expected unsent receipt to have an ID")
	}
}

func TestStore_ListAvailableForSync_CrossOrg(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	sent := saveReceipt(t, store, ctx, orgID, "100003", true)

	docs, err := store.ListAvailableForSync(ctx, 999)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Fatalf("expected empty queue for other org, got %d", len(docs))
	}
	if sent.ID == 0 {
		t.Fatal("expected receipt to have an ID")
	}
}

func TestStore_SynchronizeByID_Assign(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := saveReceipt(t, store, ctx, orgID, "100010", true)

	status := "Отгружен"
	uuid := "1c-doc-010"

	result, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, UUID: &uuid, Status: &status},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 {
		t.Fatalf("expected Updated 1, got %d", result.Updated)
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.UUID != uuid {
		t.Fatalf("expected UUID %s, got %s", uuid, doc.Receipt.UUID)
	}
	if doc.Receipt.Status != status {
		t.Fatalf("expected Status %s, got %s", status, doc.Receipt.Status)
	}

	remaining, err := store.ListAvailableForSync(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected empty queue, got %d", len(remaining))
	}
}

func TestStore_SynchronizeByID_PartialStatus(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := saveReceipt(t, store, ctx, orgID, "100011", true)

	status := "Проверен"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, Status: &status},
	}); err != nil {
		t.Fatal(err)
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.Status != status {
		t.Fatalf("expected Status %s, got %s", status, doc.Receipt.Status)
	}
	if doc.Receipt.UUID != "" {
		t.Fatalf("expected UUID to stay empty, got %s", doc.Receipt.UUID)
	}
}

func TestStore_SynchronizeByID_UUIDImmutable(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := saveReceipt(t, store, ctx, orgID, "100012", true)

	uuid1 := "uuid-x"
	uuid2 := "uuid-y"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, UUID: &uuid1},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, UUID: &uuid2},
	}); err != ErrUUIDAlreadyAssigned {
		t.Fatalf("expected ErrUUIDAlreadyAssigned, got %v", err)
	}
}

func TestStore_SynchronizeByID_IdempotentSameUUID(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := saveReceipt(t, store, ctx, orgID, "100013", true)

	uuid := "uuid-immutable"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, UUID: &uuid},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, UUID: &uuid},
	}); err != nil {
		t.Fatalf("expected same UUID to be idempotent, got %v", err)
	}
}

func TestStore_SynchronizeByID_NotFound(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	status := "test"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: 999999, Status: &status},
	}); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_UpdateByExternal_Partial(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := saveReceipt(t, store, ctx, orgID, "100020", true)
	uuid := "1c-status-020"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, UUID: &uuid},
	}); err != nil {
		t.Fatal(err)
	}

	status := "Оплачен"
	if err := store.UpdateByExternal(ctx, orgID, uuid, &status); err != nil {
		t.Fatal(err)
	}

	doc, err := store.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.Status != status {
		t.Fatalf("expected Status %s, got %s", status, doc.Receipt.Status)
	}
}

func TestStore_UpdateByExternal_NotFound(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	status := "test"
	if err := store.UpdateByExternal(ctx, orgID, "no-such-uuid", &status); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_UpdateByExternal_CrossOrgNotFound(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	rec := saveReceipt(t, store, ctx, orgID, "100021", true)
	uuid := "1c-status-021"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{
		{ID: rec.ID, UUID: &uuid},
	}); err != nil {
		t.Fatal(err)
	}

	status := "test"
	if err := store.UpdateByExternal(ctx, 999, uuid, &status); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for other org, got %v", err)
	}
}
