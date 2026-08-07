package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func TestReceiptSubmit_FullCycle(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	prodStore := products.NewStore(db)
	prodID, _ := insertProduct(t, db, orgID, "Test Product", "pcs")

	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: orgs,
		products:      prodStore,
	}

	// 1. Create via ReceiptSave (id=0)
	body := "number=001&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&total=1000&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][unit]=pcs" +
		"&items[0][quantity]=2" +
		"&items[0][price]=500" +
		"&items[0][amount]=1000"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("step 1: expected 303, got %d: %s", w.Code, w.Body.String())
	}

	if loc := w.Header().Get("Location"); loc != "/receipts" {
		t.Fatalf("step 1: expected redirect to list, got %s", loc)
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("step 1: expected one saved receipt, got %d: %v", len(list), err)
	}
	id := list[0].ID
	idStr := strconv.FormatInt(id, 10)

	// 2. GET /receipts/{id} — page renders with submit button
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/receipts/"+idStr, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("step 2: expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `class="card-header"`) || !strings.Contains(body, `class="card-content"`) {
		t.Fatal("expected receipt card to use the shared card component")
	}
	if strings.Contains(w.Body.String(), "page-card") {
		t.Fatal("unexpected legacy receipt card markup")
	}
	if !strings.Contains(w.Body.String(), "Отправить") {
		t.Fatal("step 2: expected submit button before send")
	}
	if !strings.Contains(w.Body.String(), `name="number"`) || !strings.Contains(w.Body.String(), `name="date"`) {
		t.Fatal("step 2: expected editable receipt form before send")
	}
	if strings.Contains(w.Body.String(), "К списку") {
		t.Fatal("step 2: unexpected list button")
	}

	// 3. POST /receipts/{id}/send — submit
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr+"/send", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSubmit(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("step 3: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/receipts" {
		t.Fatalf("step 3: expected redirect to list, got %s", loc)
	}

	// 4. GET /receipts/{id} — button should be gone
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/receipts/"+idStr, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("step 4: expected 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "Отправить") {
		t.Fatal("step 4: expected no submit button after send")
	}
	if strings.Contains(w.Body.String(), `name="number"`) {
		t.Fatal("step 4: expected read-only receipt after send")
	}

	// 5. POST /receipts/{id}/send again — should fail
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr+"/send", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSubmit(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("step 5: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// 6. POST /receipts/{id} to update — should fail
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr,
		strings.NewReader("number=002"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSave(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("step 6: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// 7. POST /receipts/{id}/delete — should fail
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr+"/delete", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptDelete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("step 7: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// 8. Store-level: SentAt must be non-nil now
	doc, err := app.receipts.GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.SentAt == nil {
		t.Fatal("step 8: expected SentAt to be set after submit")
	}
}

func TestReceiptSave_HtmxValidationError(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	body := "organization_id=" + strconv.FormatInt(orgID, 10) + "&user_id=1&customer_id=0"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	var resp ValidationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Title != "Ошибка документа" {
		t.Errorf("expected title 'Ошибка документа', got %q", resp.Title)
	}
	if !slices.Contains(resp.Errors, "Выберите клиента") {
		t.Errorf("expected 'Выберите клиента' in errors, got %#v", resp.Errors)
	}
	if resp.Fields["customer_id"] != "Выберите клиента" {
		t.Errorf("expected fields.customer_id, got %#v", resp.Fields)
	}
}

func TestReceiptSave_ValidationFullPage(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	body := "organization_id=" + strconv.FormatInt(orgID, 10) + "&user_id=1&customer_id=0"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Выберите клиента") {
		t.Fatal("expected inline 'Выберите клиента' on full page")
	}
}

func TestReceiptSend_HtmxEmptyItems(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	body := "organization_id=" + strconv.FormatInt(orgID, 10) + "&user_id=1&customer_id=1&send_to_1c=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ValidationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !slices.Contains(resp.Errors, "Добавьте хотя бы одну позицию") {
		t.Errorf("expected 'Добавьте хотя бы одну позицию' in errors, got %#v", resp.Errors)
	}
	if strings.Contains(w.Body.String(), "fields") {
		t.Errorf("expected no fields key for generic error, got %s", w.Body.String())
	}
}

func TestReceiptSend_ValidationFullPageEmptyItems(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	body := "organization_id=" + strconv.FormatInt(orgID, 10) + "&user_id=1&customer_id=1&send_to_1c=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Добавьте хотя бы одну позицию") {
		t.Fatal("expected inline 'Добавьте хотя бы одну позицию' on full page")
	}
}

func TestReceiptSave_EmptyItems(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	body := "number=001&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&total=0&date=2026-07-29"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected one saved receipt, got %d: %v", len(list), err)
	}
	doc, err := app.receipts.GetByID(context.Background(), list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(doc.Items))
	}
}

func TestReceiptSave_ExistingRemoveAllItems(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")
	prodStore := products.NewStore(db)
	prodID, _ := insertProduct(t, db, orgID, "Test Product", "pcs")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}

	body := "number=001&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&total=1000&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=2&items[0][price]=500&items[0][amount]=1000"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("create: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	list, _ := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	id := list[0].ID
	idStr := strconv.FormatInt(id, 10)

	body = "number=001&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&total=0&date=2026-07-29"
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSave(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update: expected 303, got %d: %s", w.Code, w.Body.String())
	}

	doc, err := app.receipts.GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Items) != 0 {
		t.Fatalf("expected 0 items after removing all, got %d", len(doc.Items))
	}
}

func TestReceiptCard_ProductNameWrapping(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")
	prodStore := products.NewStore(db)

	names := []string{
		"Монитор LG UltraGear 24GS60F-B",
		"Гайка M10 DIN 934 оцинкованная",
		"ОченьОченьОченьОченьОченьОченьДлинноеСлово",
	}
	prodIDs := make([]int64, len(names))
	for i, n := range names {
		id, _ := insertProduct(t, db, orgID, n, "шт")
		prodIDs[i] = id
	}

	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}

	var b strings.Builder
	b.WriteString("number=001&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&total=300&date=2026-07-29")
	for i, pid := range prodIDs {
		b.WriteString("&items[" + strconv.Itoa(i) + "][product_id]=" + strconv.FormatInt(pid, 10))
		b.WriteString("&items[" + strconv.Itoa(i) + "][quantity]=1")
		b.WriteString("&items[" + strconv.Itoa(i) + "][price]=100")
		b.WriteString("&items[" + strconv.Itoa(i) + "][amount]=100")
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(b.String()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("create: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("create: expected one receipt, got %d: %v", len(list), err)
	}
	idStr := strconv.FormatInt(list[0].ID, 10)

	// send so the read-only view (SSR) with lines renders
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr+"/send", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSubmit(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("send: expected 303, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/receipts/"+idStr, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("render: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, n := range names {
		if !strings.Contains(body, n) {
			t.Fatalf("expected product name %q in rendered card", n)
		}
	}
	for _, cls := range []string{
		`class="receipt-item-main"`,
		`class="receipt-item-unit-inline"`,
		`class="receipt-item-field receipt-item-qty"`,
		`class="receipt-item-field receipt-item-price"`,
		`class="receipt-item-field receipt-item-amount"`,
	} {
		if !strings.Contains(body, cls) {
			t.Fatalf("expected %s in rendered card", cls)
		}
	}
	if !strings.Contains(body, "Количество") {
		t.Fatal("expected full 'Количество' label text preserved in template")
	}
}

func TestReceiptsListPage_BlankPageRegression(t *testing.T) {
	app := &App{
		receipts: receipts.NewStore(testutil.NewTestDB(t, NewSchema())),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty body")
	}
	if !strings.Contains(body, "Товарные чеки") {
		t.Fatalf("expected list title in body")
	}
	if !strings.Contains(body, "/static/favicon.ico") {
		t.Fatalf("expected favicon link in rendered layout")
	}
}

func insertProduct(t *testing.T, dbt *sql.DB, orgID int64, name, unit string) (int64, string) {
	t.Helper()
	uuid := "uuid-" + name
	res, err := dbt.Exec(`
		INSERT INTO products (uuid, organization_id, name, unit, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid, orgID, name, unit)
	if err != nil {
		t.Fatalf("insert product %s: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id, name
}
