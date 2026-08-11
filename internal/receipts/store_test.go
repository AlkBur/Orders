package receipts

import (
	"context"
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
