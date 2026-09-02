package receipts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/entity"
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
	if err := schema.Register(ActionsTable); err != nil {
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

func saveReceiptOnDate(t *testing.T, store *Store, ctx context.Context, orgID int64, date, number string) *Receipt {
	t.Helper()
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatal(err)
	}
	rec := &Receipt{
		Number:         number,
		Date:           d,
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          100,
		Status:         "",
	}
	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}
	return rec
}

// TestStore_ListPage_PagesMatchFullList проверяет, что keyset-пагинация
// не теряет и не дублирует документы: последовательность порций должна
// совпадать с полной выборкой List (эталон). Порядок — id DESC.
func TestStore_ListPage_PagesMatchFullList(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	dates := []string{"2026-08-05", "2026-08-04", "2026-08-03", "2026-08-02", "2026-08-01"}
	for i, d := range dates {
		saveReceiptOnDate(t, store, ctx, orgID, d, "p"+itoa(i))
	}

	all, err := store.List(ctx, ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 receipts, got %d", len(all))
	}

	var collected []*Receipt
	var after *Cursor
	for {
		page, err := store.ListPage(ctx, ListOptions{Limit: 2, After: after}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) == 0 {
			t.Fatal("page returned no items before HasMore=false")
		}
		if len(page.Items) > 2 {
			t.Fatalf("page larger than limit: %d", len(page.Items))
		}
		collected = append(collected, page.Items...)
		if !page.HasMore {
			if page.Next != nil {
				t.Fatal("Next must be nil when HasMore is false")
			}
			break
		}
		if page.Next == nil {
			t.Fatal("Next must be set when HasMore is true")
		}
		after = page.Next
	}

	if len(collected) != len(all) {
		t.Fatalf("expected %d collected, got %d", len(all), len(collected))
	}
	for i := range all {
		if collected[i].ID != all[i].ID {
			t.Fatalf("mismatch at %d: got id %d, want id %d", i, collected[i].ID, all[i].ID)
		}
	}
}

// TestStore_ListPage_IDCursor проверяет курсор по id: порции разбивают
// выборку по убыванию id, одинаковые даты не влияют на порядок.
func TestStore_ListPage_IDCursor(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	for i := 1; i <= 5; i++ {
		saveReceiptOnDate(t, store, ctx, orgID, "2026-08-01", "d"+itoa(i))
	}

	var ids []int64
	var after *Cursor
	for {
		page, err := store.ListPage(ctx, ListOptions{Limit: 2, After: after}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range page.Items {
			ids = append(ids, r.ID)
		}
		if !page.HasMore {
			break
		}
		after = page.Next
	}

	want := []int64{5, 4, 3, 2, 1}
	if len(ids) != len(want) {
		t.Fatalf("expected %d ids, got %d", len(want), len(ids))
	}
	for i, id := range want {
		if ids[i] != id {
			t.Fatalf("position %d: got id %d, want %d", i, ids[i], id)
		}
	}
}

// TestStore_ListPage_LastPageHasNoNext проверяет, что последняя порция
// помечается HasMore=false и не порождает следующий запрос.
func TestStore_ListPage_LastPageHasNoNext(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)
	for i := 1; i <= 3; i++ {
		saveReceiptOnDate(t, store, ctx, orgID, "2026-08-0"+itoa(i), "n"+itoa(i))
	}

	first, err := store.ListPage(ctx, ListOptions{Limit: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !first.HasMore {
		t.Fatal("expected HasMore on non-last page")
	}

	last, err := store.ListPage(ctx, ListOptions{Limit: 2, After: first.Next}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if last.HasMore {
		t.Fatal("expected HasMore=false on last page")
	}
	if last.Next != nil {
		t.Fatal("expected Next=nil on last page")
	}
}

// TestStore_ListPage_LimitOne проверяет граничное значение limit=1.
func TestStore_ListPage_LimitOne(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)
	for i := 1; i <= 3; i++ {
		saveReceiptOnDate(t, store, ctx, orgID, "2026-08-0"+itoa(i), "m"+itoa(i))
	}

	page, err := store.ListPage(ctx, ListOptions{Limit: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
	}
	if !page.HasMore {
		t.Fatal("expected HasMore with limit=1 and more rows")
	}
}

// TestStore_ListPage_QueryAppliedPerPage проверяет, что текстовый поиск
// применяется к каждой порции, в том числе вместе с курсором, взятым
// из другой выборки: чужие документы не просачиваются.
func TestStore_ListPage_QueryAppliedPerPage(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	// Перемешиваем даты и номера: alpha/beta на одних и тех же датах.
	rows := []struct{ date, number string }{
		{"2026-08-05", "alpha-1"},
		{"2026-08-05", "beta-1"},
		{"2026-08-04", "alpha-2"},
		{"2026-08-04", "beta-2"},
		{"2026-08-03", "alpha-3"},
		{"2026-08-03", "beta-3"},
	}
	for _, r := range rows {
		saveReceiptOnDate(t, store, ctx, orgID, r.date, r.number)
	}

	visible := entity.Names(Descriptor.ListFields())

	alpha, err := store.ListPage(ctx, ListOptions{Limit: 2, Query: "alpha", Filter: Filter{}}, visible)
	if err != nil {
		t.Fatal(err)
	}
	if alpha.Next == nil {
		t.Fatal("expected alpha page to have a next cursor")
	}

	// Курсор из выборки alpha применяется к выборке beta: возвращённые
	// документы обязаны быть beta (WHERE применяется до курсора).
	beta, err := store.ListPage(ctx, ListOptions{Limit: 10, Query: "beta", After: alpha.Next}, visible)
	if err != nil {
		t.Fatal(err)
	}
	if len(beta.Items) == 0 {
		t.Fatal("expected beta rows after alpha cursor")
	}
	for _, r := range beta.Items {
		if len(r.Number) < 5 || r.Number[:4] != "beta" {
			t.Fatalf("foreign document leaked into beta selection: %q", r.Number)
		}
	}
}

// TestStore_ListPage_LastPageDoesNotProduceNextQuery — результат последней
// порции не создаёт следующий запрос (HasMore=false → сервер не выводит
// sentinel). Дублируется в app-тестах; здесь проверяется контракт хранилища.
func TestStore_ListPage_SinglePageNoHasMore(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)
	saveReceiptOnDate(t, store, ctx, orgID, "2026-08-01", "solo")

	page, err := store.ListPage(ctx, ListOptions{Limit: 50}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
	}
	if page.HasMore {
		t.Fatal("expected HasMore=false when rows <= limit")
	}
	if page.Next != nil {
		t.Fatal("expected Next=nil when rows <= limit")
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

// saveReceiptWith создаёт чек с управляемыми полями для тестов фильтра.
func saveReceiptWith(t *testing.T, store *Store, ctx context.Context, number string, date time.Time, total float64, orgID, custID int64, status string, sent bool) *Receipt {
	t.Helper()
	rec := &Receipt{
		Number:         number,
		Date:           date,
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     custID,
		Total:          total,
		Status:         status,
	}
	if sent {
		now := time.Now()
		rec.SentAt = &now
	}
	if err := store.Save(ctx, &Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}
	return rec
}

func fptr(v float64) *float64 { return &v }

func TestStore_List_FilterByDateAndAmount(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)

	saveReceiptWith(t, store, ctx, "100101", time.Date(2026, 1, 5, 12, 0, 0, 0, time.Local), 100, orgID, custID, "", false)
	saveReceiptWith(t, store, ctx, "100102", time.Date(2026, 2, 10, 12, 0, 0, 0, time.Local), 250, orgID, custID, "", false)
	saveReceiptWith(t, store, ctx, "100103", time.Date(2026, 3, 15, 12, 0, 0, 0, time.Local), 400, orgID, custID, "", false)

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"no filter", Filter{}, 3},
		{"date_from", Filter{DateFrom: "2026-02-01"}, 2},
		{"date_to", Filter{DateTo: "2026-02-28"}, 2},
		{"date_from_and_to", Filter{DateFrom: "2026-02-01", DateTo: "2026-02-28"}, 1},
		{"amount_from", Filter{AmountFrom: fptr(150)}, 2},
		{"amount_to", Filter{AmountTo: fptr(250)}, 2},
		{"amount_range", Filter{AmountFrom: fptr(100), AmountTo: fptr(250)}, 2},
		{"date_and_amount", Filter{DateFrom: "2026-02-01", AmountTo: fptr(250)}, 1},
	}
	for _, c := range cases {
		list, err := store.List(ctx, ListOptions{Filter: c.f}, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(list) != c.want {
			t.Errorf("%s: got %d receipts, want %d", c.name, len(list), c.want)
		}
	}
}

func TestStore_List_FilterByOrgAndCustomer(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)

	orgStore := organizations.NewStore(store.db)
	org2 := orgStore.New()
	org2.UUID = "rec-filter-org2"
	org2.Name = "Filter Org 2"
	org2.APIKey = "rec-filter-key2"
	if err := orgStore.Save(ctx, org2); err != nil {
		t.Fatal(err)
	}

	custStore := customers.NewStore(store.db)
	cust2 := custStore.New()
	cust2.UUID = "rec-filter-cust2"
	cust2.Name = "Filter Customer 2"
	cust2.OrganizationID = org2.ID
	if err := custStore.Save(ctx, cust2); err != nil {
		t.Fatal(err)
	}

	saveReceiptWith(t, store, ctx, "100201", time.Date(2026, 1, 5, 12, 0, 0, 0, time.Local), 100, orgID, custID, "", false)
	saveReceiptWith(t, store, ctx, "100202", time.Date(2026, 2, 10, 12, 0, 0, 0, time.Local), 250, org2.ID, cust2.ID, "", false)
	saveReceiptWith(t, store, ctx, "100203", time.Date(2026, 3, 15, 12, 0, 0, 0, time.Local), 400, orgID, cust2.ID, "", false)

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"org", Filter{OrganizationID: org2.ID}, 1},
		{"customer", Filter{CustomerID: cust2.ID}, 2},
		{"org_and_customer", Filter{OrganizationID: org2.ID, CustomerID: cust2.ID}, 1},
	}
	for _, c := range cases {
		list, err := store.List(ctx, ListOptions{Filter: c.f}, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(list) != c.want {
			t.Errorf("%s: got %d receipts, want %d", c.name, len(list), c.want)
		}
	}
}

func TestStore_List_FilterByStatus(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)

	// Свежий «Создан» (пустой статус, created_at = now).
	saveReceiptWith(t, store, ctx, "100301", time.Date(2026, 1, 5, 12, 0, 0, 0, time.Local), 100, orgID, custID, "", false)
	// «Создан», но просрочен: created_at старится вручную.
	overdue := saveReceiptWith(t, store, ctx, "100302", time.Date(2026, 1, 6, 12, 0, 0, 0, time.Local), 200, orgID, custID, "", false)
	if _, err := store.db.Exec(`UPDATE receipts SET created_at = datetime('now', '-2 days') WHERE id = ?`, overdue.ID); err != nil {
		t.Fatal(err)
	}
	// «Отправлен» (пустой статус + sent_at).
	saveReceiptWith(t, store, ctx, "100303", time.Date(2026, 1, 7, 12, 0, 0, 0, time.Local), 300, orgID, custID, "", true)
	// «Принят» (статус пришёл из 1С).
	saveReceiptWith(t, store, ctx, "100304", time.Date(2026, 1, 8, 12, 0, 0, 0, time.Local), 400, orgID, custID, StatusAccepted, false)

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"created excludes overdue", Filter{Status: StatusCreated}, 1},
		{"overdue", Filter{Status: OverdueStatus}, 1},
		{"sent", Filter{Status: StatusSent}, 1},
		{"accepted", Filter{Status: StatusAccepted}, 1},
		{"unknown status is ignored", Filter{Status: "Неизвестный"}, 4},
	}
	for _, c := range cases {
		list, err := store.List(ctx, ListOptions{Filter: c.f}, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(list) != c.want {
			t.Errorf("%s: got %d receipts, want %d", c.name, len(list), c.want)
		}
	}
}

func TestReceiptDeletable(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"", true},
		{StatusCreated, true},
		{StatusSent, true},
		{StatusCancelled, true},
		{StatusAccepted, false},
		{StatusProcessed, false},
		{StatusFinished, false},
		{OverdueStatus, false},
	}
	for _, tt := range tests {
		if got := ReceiptDeletable(tt.status); got != tt.want {
			t.Errorf("ReceiptDeletable(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestStore_MarkDeleted_StatusMatrix(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)

	tests := []struct {
		status  string
		sent    bool
		wantErr error
	}{
		{"", false, nil},
		{StatusCreated, false, nil},
		{StatusSent, true, nil},
		{StatusCancelled, false, nil},
		{StatusAccepted, true, ErrReceiptNotDeletable},
		{StatusProcessed, true, ErrReceiptNotDeletable},
		{StatusFinished, true, ErrReceiptNotDeletable},
	}
	for i, tt := range tests {
		num := fmt.Sprintf("MD%03d", i)
		rec := saveReceiptWith(t, store, ctx, num, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), 100, orgID, custID, tt.status, tt.sent)

		err := store.MarkDeleted(ctx, rec.ID)
		if tt.wantErr == nil {
			if err != nil {
				t.Fatalf("status %q: MarkDeleted error = %v, want nil", tt.status, err)
			}
			// Помеченный документ должен исчезнуть из бизнес-чтений.
			if _, err := store.GetByID(ctx, rec.ID); err != ErrNotFound {
				t.Fatalf("status %q: GetByID after mark = %v, want ErrNotFound", tt.status, err)
			}
		} else {
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("status %q: MarkDeleted error = %v, want %v", tt.status, err, tt.wantErr)
			}
		}
	}
}

func TestStore_MarkDeleted_NotFound(t *testing.T) {
	ctx, store, _, _ := setupTestData(t)
	if err := store.MarkDeleted(ctx, 99999); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_MarkDeleted_Repeat(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "MDREP", time.Now(), 100, orgID, custID, StatusCreated, false)

	if err := store.MarkDeleted(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	// После пометки у документа статус StatusCancelled. Приоритет имеет
	// признак deleted_at: повторная пометка возвращает ErrNotFound.
	if err := store.MarkDeleted(ctx, rec.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound on repeat mark, got %v", err)
	}
}

func TestStore_MarkDeleted_SetsFields(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "MDFIELDS", time.Now(), 100, orgID, custID, StatusSent, true)

	if err := store.MarkDeleted(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}

	var deletedAt sql.NullTime
	var status string
	var sentAt sql.NullTime
	err := store.db.QueryRowContext(ctx,
		`SELECT deleted_at, status, sent_at FROM receipts WHERE id = ?`, rec.ID).
		Scan(&deletedAt, &status, &sentAt)
	if err != nil {
		t.Fatal(err)
	}
	if !deletedAt.Valid {
		t.Fatal("expected deleted_at to be set after mark")
	}
	if status != StatusCancelled {
		t.Fatalf("expected status %q, got %q", StatusCancelled, status)
	}
	if sentAt.Valid {
		t.Fatal("expected sent_at to be NULL after mark")
	}
}

func TestStore_MarkDeleted_HidesDocument(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	active := saveReceiptWith(t, store, ctx, "MDACTIVE", time.Now(), 100, orgID, custID, StatusCreated, false)
	sent := saveReceiptWith(t, store, ctx, "MDSENT", time.Now(), 100, orgID, custID, StatusSent, true)
	extUUID := "ext-1c"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{{ID: sent.ID, UUID: &extUUID}}); err != nil {
		t.Fatal(err)
	}

	before, err := store.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkDeleted(ctx, sent.ID); err != nil {
		t.Fatal(err)
	}

	if got, err := store.Count(ctx); err != nil || got != before-1 {
		t.Fatalf("Count() = %d, want %d (%v)", got, before-1, err)
	}
	if _, err := store.GetByID(ctx, sent.ID); err != ErrNotFound {
		t.Fatalf("GetByID marked: got %v, want ErrNotFound", err)
	}
	if _, err := store.GetByExternal(ctx, extUUID); err != ErrNotFound {
		t.Fatalf("GetByExternal marked: got %v, want ErrNotFound", err)
	}

	all, err := store.List(ctx, ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range all {
		if r.ID == sent.ID {
			t.Fatal("marked receipt still present in List")
		}
	}

	queue, err := store.ListAvailableForSync(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range queue {
		if d.Receipt.ID == sent.ID {
			t.Fatal("marked receipt still present in sync queue")
		}
	}

	// Непомеченный документ остаётся видимым.
	if _, err := store.GetByID(ctx, active.ID); err != nil {
		t.Fatalf("active receipt unexpectedly hidden: %v", err)
	}
}

func TestStore_MarkDeleted_BlocksIntegration(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "MDBLOCK", time.Now(), 100, orgID, custID, StatusCreated, false)
	extUUID := "ext-block"
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{{ID: rec.ID, UUID: &extUUID}}); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkDeleted(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}

	status := StatusAccepted
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{{ID: rec.ID, Status: &status}}); err != ErrNotFound {
		t.Fatalf("SynchronizeByID marked: got %v, want ErrNotFound", err)
	}
	if err := store.UpdateByExternal(ctx, orgID, extUUID, &status); err != ErrNotFound {
		t.Fatalf("UpdateByExternal marked: got %v, want ErrNotFound", err)
	}
	// Legacy-путь обновления по exchange_id также отсекает помеченные.
	legacyUUID := "legacy-uuid"
	if err := store.Synchronize(ctx, []ReceiptUpdate{{ExchangeID: rec.ExchangeID, UUID: &legacyUUID}}); err != ErrExchangeIDNotFound {
		t.Fatalf("Synchronize marked: got %v, want ErrExchangeIDNotFound", err)
	}
}

func assignUUID(t *testing.T, store *Store, ctx context.Context, orgID int64, rec *Receipt, uuid string) {
	t.Helper()
	if _, err := store.SynchronizeByID(ctx, orgID, []SyncUpdate{{ID: rec.ID, UUID: &uuid}}); err != nil {
		t.Fatal(err)
	}
	rec.UUID = uuid
}

func TestStore_SetAction_Upsert(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT001", time.Now(), 100, orgID, custID, StatusCreated, false)

	if err := store.SetAction(ctx, rec.ID, ActionDelete); err != nil {
		t.Fatal(err)
	}
	action, receivedAt, err := store.GetAction(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionDelete {
		t.Fatalf("expected action %q, got %q", ActionDelete, action)
	}
	if receivedAt != nil {
		t.Fatal("expected action_received_at to be nil after set")
	}

	// Повторный выбор меняет действие и сбрасывает дату получения.
	if err := store.SetAction(ctx, rec.ID, ActionChange); err != nil {
		t.Fatal(err)
	}
	action, receivedAt, err = store.GetAction(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionChange {
		t.Fatalf("expected action %q, got %q", ActionChange, action)
	}
	if receivedAt != nil {
		t.Fatal("expected action_received_at to be nil after re-set")
	}
}

func TestStore_SetAction_Clear(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT002", time.Now(), 100, orgID, custID, StatusCreated, false)

	if err := store.SetAction(ctx, rec.ID, ActionDelete); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAction(ctx, rec.ID, ""); err != nil {
		t.Fatal(err)
	}

	action, receivedAt, err := store.GetAction(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if action != "" {
		t.Fatalf("expected empty action after clear, got %q", action)
	}
	if receivedAt != nil {
		t.Fatal("expected nil receivedAt after clear")
	}

	// Строка сохраняется, а не удаляется.
	var n int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM receipt_actions WHERE receipt_id = ?`, rec.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected action row to persist after clear, got %d", n)
	}
}

func TestStore_SetAction_Clear_WithoutRow(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT002B", time.Now(), 100, orgID, custID, StatusCreated, false)

	// Отмена без предварительной установки действия ничего не создаёт.
	if err := store.SetAction(ctx, rec.ID, ""); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM receipt_actions WHERE receipt_id = ?`, rec.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected no row after clear without prior action, got %d", n)
	}
	action, receivedAt, err := store.GetAction(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if action != "" || receivedAt != nil {
		t.Fatalf("expected no action, got %q / %v", action, receivedAt)
	}
}

func TestStore_GetAction_None(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT003", time.Now(), 100, orgID, custID, StatusCreated, false)

	action, receivedAt, err := store.GetAction(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if action != "" || receivedAt != nil {
		t.Fatalf("expected no action, got %q / %v", action, receivedAt)
	}
}

func TestStore_ListActionsForSync(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec1 := saveReceiptWith(t, store, ctx, "ACT004", time.Now(), 100, orgID, custID, StatusCreated, false)
	rec2 := saveReceiptWith(t, store, ctx, "ACT005", time.Now(), 100, orgID, custID, StatusCreated, false)
	assignUUID(t, store, ctx, orgID, rec1, "act-004")
	assignUUID(t, store, ctx, orgID, rec2, "act-005")

	if err := store.SetAction(ctx, rec1.ID, ActionChange); err != nil {
		t.Fatal(err)
	}

	actions, err := store.ListActionsForSync(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 pending action, got %d", len(actions))
	}
	if actions[0].UUID != "act-004" || actions[0].Action != ActionChange {
		t.Fatalf("unexpected action item: %+v", actions[0])
	}
}

func TestStore_ListActionsForSync_Empty(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	actions, err := store.ListActionsForSync(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected empty actions, got %d", len(actions))
	}
}

func TestStore_ListActionsForSync_Cancelled(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT005B", time.Now(), 100, orgID, custID, StatusCreated, false)
	assignUUID(t, store, ctx, orgID, rec, "act-005b")

	if err := store.SetAction(ctx, rec.ID, ActionDelete); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAction(ctx, rec.ID, ""); err != nil {
		t.Fatal(err)
	}

	// Отменённое действие попадает в очередь с пустым action.
	actions, err := store.ListActionsForSync(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 cancelled action in queue, got %d", len(actions))
	}
	if actions[0].UUID != "act-005b" || actions[0].Action != "" {
		t.Fatalf("unexpected cancelled action item: %+v", actions[0])
	}

	// После подтверждения исчезает из очереди.
	if _, err := store.ConfirmActions(ctx, orgID, []string{"act-005b"}); err != nil {
		t.Fatal(err)
	}
	actions, err = store.ListActionsForSync(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected empty queue after confirm, got %d", len(actions))
	}
}

func TestStore_ConfirmActions(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT006", time.Now(), 100, orgID, custID, StatusCreated, false)
	assignUUID(t, store, ctx, orgID, rec, "act-006")

	if err := store.SetAction(ctx, rec.ID, ActionDelete); err != nil {
		t.Fatal(err)
	}

	result, err := store.ConfirmActions(ctx, orgID, []string{"act-006"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", result.Updated)
	}

	_, receivedAt, err := store.GetAction(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if receivedAt == nil {
		t.Fatal("expected action_received_at to be set after confirm")
	}

	// Повторное подтверждение идемпотентно.
	result, err = store.ConfirmActions(ctx, orgID, []string{"act-006"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 0 {
		t.Fatalf("expected 0 updated on repeat, got %d", result.Updated)
	}
}

func TestStore_ConfirmActions_NotFound(t *testing.T) {
	ctx, store, orgID, _ := setupTestData(t)

	if _, err := store.ConfirmActions(ctx, orgID, []string{"missing-uuid"}); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_MarkDeleted_RemovesAction(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT007", time.Now(), 100, orgID, custID, StatusSent, true)
	assignUUID(t, store, ctx, orgID, rec, "act-007")

	if err := store.SetAction(ctx, rec.ID, ActionDelete); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkDeleted(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM receipt_actions WHERE receipt_id = ?`, rec.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected action row removed on soft delete, got %d", n)
	}
}

func TestStore_DeleteByID_RemovesAction(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT008", time.Now(), 100, orgID, custID, StatusCreated, false)

	if err := store.SetAction(ctx, rec.ID, ActionChange); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteByID(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM receipt_actions WHERE receipt_id = ?`, rec.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected action row removed on physical delete, got %d", n)
	}
}

func TestStore_List_FilterByActionSetAt(t *testing.T) {
	ctx, store, orgID, custID := setupTestData(t)
	rec := saveReceiptWith(t, store, ctx, "ACT009", time.Now(), 100, orgID, custID, StatusCreated, false)
	if err := store.SetAction(ctx, rec.ID, ActionDelete); err != nil {
		t.Fatal(err)
	}
	// Фиксируем дату установки для детерминированного отбора.
	if _, err := store.db.Exec(`UPDATE receipt_actions SET action_set_at = '2026-05-15 10:00:00' WHERE receipt_id = ?`, rec.ID); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"no filter", Filter{}, 1},
		{"from_in_range", Filter{ActionSetFrom: "2026-05-15"}, 1},
		{"to_in_range", Filter{ActionSetTo: "2026-05-15"}, 1},
		{"from_after", Filter{ActionSetFrom: "2026-05-16"}, 0},
		{"to_before", Filter{ActionSetTo: "2026-05-14"}, 0},
	}
	for _, c := range cases {
		list, err := store.List(ctx, ListOptions{Filter: c.f}, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(list) != c.want {
			t.Errorf("%s: got %d receipts, want %d", c.name, len(list), c.want)
		}
	}
}
