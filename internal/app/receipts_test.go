package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/testutil"
	"Orders/internal/users"

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

	// 7. POST /receipts/{id}/delete — mark for deletion (admin op, status Отправлен allowed)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr+"/delete", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptMarkDeleted(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("step 7: expected 303 after mark, got %d: %s", w.Code, w.Body.String())
	}

	// 8. Marked document is hidden and no longer returned by business reads
	if _, err := app.receipts.GetByID(context.Background(), id); err != receipts.ErrNotFound {
		t.Fatalf("step 8: expected ErrNotFound for marked receipt, got %v", err)
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

	// Пустой документ не сохраняется: возвращаются ошибки, редиректа нет.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if redirect := w.Header().Get("HX-Redirect"); redirect != "" {
		t.Errorf("expected no HX-Redirect, got %q", redirect)
	}
	if !strings.Contains(w.Body.String(), "Добавьте хотя бы одну строку.") {
		t.Fatalf("expected empty-document error, got %s", w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no saved receipt, got %d", len(list))
	}
}

func TestReceiptSend_FullPageEmptyItemsRejected(t *testing.T) {
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
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no saved receipt, got %d", len(list))
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

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no saved receipt, got %d", len(list))
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

	// Удаление всех строк и сохранение пустого документа отклоняется,
	// прежние строки в БД остаются без изменений.
	body = "number=001&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&total=0&date=2026-07-29"
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSave(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("update: expected 422, got %d: %s", w.Code, w.Body.String())
	}

	doc, err := app.receipts.GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Items) != 1 {
		t.Fatalf("expected original item to remain, got %d items", len(doc.Items))
	}
	if doc.Items[0].Quantity != 2 || doc.Items[0].Price != 500 || doc.Items[0].Amount != 1000 {
		t.Fatalf("expected original values, got qty=%v price=%v amount=%v",
			doc.Items[0].Quantity, doc.Items[0].Price, doc.Items[0].Amount)
	}
}

func TestReceiptSave_ZeroValuesRejected(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ZeroOrg", "kzero")
	prodStore := products.NewStore(db)
	prodID, _ := insertProduct(t, db, orgID, "Zero Product", "pcs")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}

	body := "number=Z1&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=0&items[0][price]=0&items[0][amount]=0"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no saved receipt, got %d", len(list))
	}
}

func TestReceiptSave_ZeroValuesRejected_Fragment(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ZeroFragOrg", "kzerofrag")
	prodStore := products.NewStore(db)
	prodID, _ := insertProduct(t, db, orgID, "Zero Frag Product", "pcs")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}

	body := "number=Z2&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=1&items[0][price]=0&items[0][amount]=0"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if redirect := w.Header().Get("HX-Redirect"); redirect != "" {
		t.Errorf("expected no HX-Redirect, got %q", redirect)
	}
	if !strings.Contains(w.Body.String(), "цена должна быть больше нуля") {
		t.Fatalf("expected price error, got %s", w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no saved receipt, got %d", len(list))
	}
}

func TestReceiptSave_NegativeValuesRejected(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "NegOrg", "kneg")
	prodStore := products.NewStore(db)
	prodID, _ := insertProduct(t, db, orgID, "Neg Product", "pcs")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}

	body := "number=N1&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=-1&items[0][price]=-1&items[0][amount]=-1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no saved receipt, got %d", len(list))
	}
}

// TestReceiptSave_InlineEditZeroRejected проверяет второй путь правки строки:
// значения изменены прямо в таблице (та же форма, документ существует).
// Документ не должен сохраниться, а прежние значения строк остаются в БД.
func TestReceiptSave_InlineEditZeroRejected(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "InlineOrg", "kinline")
	prodStore := products.NewStore(db)
	prodID, _ := insertProduct(t, db, orgID, "Inline Product", "pcs")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}

	create := "number=I1&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=2&items[0][price]=500&items[0][amount]=1000"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(create))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("create: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	list, _ := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	id := list[0].ID
	idStr := strconv.FormatInt(id, 10)

	update := "number=I1&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=2&items[0][price]=0&items[0][amount]=0"
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts/"+idStr, strings.NewReader(update))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSave(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("update: expected 422, got %d: %s", w.Code, w.Body.String())
	}

	doc, err := app.receipts.GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(doc.Items))
	}
	if doc.Items[0].Quantity != 2 || doc.Items[0].Price != 500 || doc.Items[0].Amount != 1000 {
		t.Fatalf("expected original item values, got qty=%v price=%v amount=%v",
			doc.Items[0].Quantity, doc.Items[0].Price, doc.Items[0].Amount)
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

// TestReceiptsList_ActionsForThreeStates — три состояния документа в
// журнале: без файлов, с файлами, отправленный. Проверяются условия
// показа кнопок «Файлы», «Отправить», «Редактировать».
func TestReceiptsList_ActionsForThreeStates(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	filesDB := testutil.NewTestDB(t, NewFilesSchema())
	orgID, _ := insertOrg(t, db, "ListOrg", "k1")

	// Чек 1: новый, без файлов. Чек 2: новый, с файлом.
	// Чек 3: отправленный, с файлом.
	insertReceiptForOrg(t, db, orgID)
	u2 := insertReceiptForOrg(t, db, orgID)
	u3 := insertReceiptForOrg(t, db, orgID)

	app := &App{
		receipts:     receipts.NewStore(db),
		receiptFiles: receipts.NewFileStore(filesDB),
	}

	doc2, _ := app.receipts.GetByExternal(context.Background(), u2)
	if _, _, err := app.receiptFiles.Upsert(context.Background(), doc2.Receipt.ID, "f2", "d2.pdf", "application/pdf", pdfBody); err != nil {
		t.Fatal(err)
	}
	doc3, _ := app.receipts.GetByExternal(context.Background(), u3)
	if _, _, err := app.receiptFiles.Upsert(context.Background(), doc3.Receipt.ID, "f3", "d3.pdf", "application/pdf", pdfBody); err != nil {
		t.Fatal(err)
	}
	// Чек 3 помечается отправленным.
	now := time.Now().Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE receipts SET sent_at = ? WHERE id = ?`, now, doc3.Receipt.ID); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	filesBtns := strings.Count(body, `title="Файлы"`)
	if filesBtns != 2 {
		t.Fatalf("expected 2 Files buttons (docs 2,3), got %d:\n%s", filesBtns, body)
	}
	sendBtns := strings.Count(body, `title="Отправить"`)
	if sendBtns != 2 {
		t.Fatalf("expected 2 Send buttons (docs 1,2), got %d", sendBtns)
	}
	editBtns := strings.Count(body, `title="Редактировать"`)
	if editBtns != 2 {
		t.Fatalf("expected 2 Edit buttons (docs 1,2), got %d", editBtns)
	}
}

// ===========================================================================
// keyset-пагинация списка чеков (lazy loading, sentinel)
// ===========================================================================

// insertReceiptRaw вставляет чек с заданным номером и датой напрямую в БД.
func insertReceiptRaw(t *testing.T, db *sql.DB, orgID int64, number, date string) int64 {
	t.Helper()
	res, err := db.Exec(`
		INSERT INTO receipts (uuid, exchange_id, number, date, organization_id,
			user_id, customer_id, total, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, "uuid-"+number, "exch-"+number, number, date, orgID)
	if err != nil {
		t.Fatalf("insert receipt %s: %v", number, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// receiptEditURLs извлекает из тела ссылки «Редактировать» — по одной
// на строку журнала. Уникальность URL используется как идентичность
// документа при проверке отсутствия дублей между порциями.
func receiptEditURLs(t *testing.T, body string) []string {
	t.Helper()
	re := regexp.MustCompile(`href="([^"]+)" title="Редактировать"`)
	ms := re.FindAllStringSubmatch(body, -1)
	urls := make([]string, 0, len(ms))
	for _, m := range ms {
		urls = append(urls, m[1])
	}
	return urls
}

// receiptLoadMoreURL извлекает URL следующей порции из sentinel
// lazy loading. Возвращает пустую строку, если sentinel отсутствует.
// URL в атрибуте HTML-экранирован (&amp;) — восстанавливаем символы.
func receiptLoadMoreURL(t *testing.T, body string) string {
	t.Helper()
	re := regexp.MustCompile(`id="receipts-load-more" hx-get="([^"]+)"`)
	m := re.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return html.UnescapeString(m[1])
}

// receiptAfterFromLoadMoreURL извлекает из URL следующей порции параметр
// after (непрозрачный курсор).
func receiptAfterFromLoadMoreURL(t *testing.T, loadMoreURL string) string {
	t.Helper()
	u, err := url.Parse(loadMoreURL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("after")
}

// TestReceiptsList_LimitDefaultsTo50 проверяет, что без параметра limit
// выводится первая порция из 50 документов и sentinel для подгрузки.
func TestReceiptsList_LimitDefaultsTo50(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "LimitOrg", "k_limit")
	app := &App{receipts: receipts.NewStore(db)}
	for i := 1; i <= 51; i++ {
		insertReceiptRaw(t, db, orgID, "l"+strconv.Itoa(i), "2026-08-01")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `id="receipts-load-more"`) {
		t.Fatal("expected sentinel when there are more than 50 receipts")
	}
	if got := len(receiptEditURLs(t, body)); got != 50 {
		t.Fatalf("expected 50 rows by default, got %d", got)
	}
}

// TestReceiptsList_NoSentinelWhenRowsFitLimit проверяет, что sentinel не
// выводится, когда все документы уместились в одной порции.
func TestReceiptsList_NoSentinelWhenRowsFitLimit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "SmallOrg", "k_small")
	app := &App{receipts: receipts.NewStore(db)}
	for i := 1; i <= 3; i++ {
		insertReceiptRaw(t, db, orgID, "s"+strconv.Itoa(i), "2026-08-01")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, `id="receipts-load-more"`) {
		t.Fatal("expected no sentinel when rows fit in one page")
	}
	if got := len(receiptEditURLs(t, body)); got != 3 {
		t.Fatalf("expected 3 rows, got %d", got)
	}
}

// TestReceiptsList_LimitParam проверяет разбор параметра limit: валидные
// значения меняют размер порции, недопустимые значения дают 400.
func TestReceiptsList_LimitParam(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "LimitParamOrg", "k_lp")
	app := &App{receipts: receipts.NewStore(db)}
	for i := 1; i <= 5; i++ {
		insertReceiptRaw(t, db, orgID, "p"+strconv.Itoa(i), "2026-08-01")
	}

	w := httptest.NewRecorder()
	app.ReceiptsPage(w, httptest.NewRequest(http.MethodGet, "/receipts?limit=1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("limit=1: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := len(receiptEditURLs(t, w.Body.String())); got != 1 {
		t.Fatalf("limit=1: expected 1 row, got %d", got)
	}

	w2 := httptest.NewRecorder()
	app.ReceiptsPage(w2, httptest.NewRequest(http.MethodGet, "/receipts?limit=100", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("limit=100: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	if got := len(receiptEditURLs(t, w2.Body.String())); got != 5 {
		t.Fatalf("limit=100: expected 5 rows, got %d", got)
	}

	for _, bad := range []string{"0", "-1", "101", "abc"} {
		w3 := httptest.NewRecorder()
		app.ReceiptsPage(w3, httptest.NewRequest(http.MethodGet, "/receipts?limit="+bad, nil))
		if w3.Code != http.StatusBadRequest {
			t.Fatalf("limit=%q: expected 400, got %d", bad, w3.Code)
		}
	}
}

// TestReceiptsList_AfterCursor проверяет keyset-пагинацию end-to-end:
// следующие порции подтягиваются через sentinel, документы не теряются
// и не дублируются.
func TestReceiptsList_AfterCursor(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "CursorOrg", "k_cursor")
	app := &App{receipts: receipts.NewStore(db)}
	for i := 1; i <= 7; i++ {
		insertReceiptRaw(t, db, orgID, "c"+strconv.Itoa(i), "2026-08-01")
	}

	var collected []string
	nextURL := "/receipts?limit=3"
	for {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, nextURL, nil)
		if strings.Contains(nextURL, "part=rows") {
			r.Header.Set("HX-Request", "true")
		}
		app.ReceiptsPage(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		collected = append(collected, receiptEditURLs(t, body)...)

		next := receiptLoadMoreURL(t, body)
		if next == "" {
			break
		}
		nextURL = next
	}

	if len(collected) != 7 {
		t.Fatalf("expected 7 rows across pages, got %d", len(collected))
	}
	seen := make(map[string]bool)
	for _, u := range collected {
		if seen[u] {
			t.Fatalf("duplicate row across pages: %s", u)
		}
		seen[u] = true
	}
}

// TestReceiptsList_FragmentRowsNoBrowser проверяет, что запрос порции
// (part=rows) возвращает только строки и sentinel, без обёртки браузера.
func TestReceiptsList_FragmentRowsNoBrowser(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "FragOrg", "k_frag")
	app := &App{receipts: receipts.NewStore(db)}
	for i := 1; i <= 2; i++ {
		insertReceiptRaw(t, db, orgID, "f"+strconv.Itoa(i), "2026-08-01")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts?part=rows&limit=2", nil)
	r.Header.Set("HX-Request", "true")
	app.ReceiptsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "receipts-row") {
		t.Fatal("expected rows fragment")
	}
	if strings.Contains(body, "receipts-browser") {
		t.Fatal("rows fragment must not include the browser container")
	}
}

// TestReceiptsList_InvalidCursor проверяет, что повреждённый или поддельный
// курсор after возвращает 400, а не молчаливую перезагрузку первой порции.
func TestReceiptsList_InvalidCursor(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "BadCursorOrg", "k_bad")
	app := &App{receipts: receipts.NewStore(db)}
	insertReceiptRaw(t, db, orgID, "b1", "2026-08-01")

	for _, after := range []string{
		"garbage",
		"###",
		encodeReceiptCursor(receipts.Cursor{ID: 0}),
	} {
		w := httptest.NewRecorder()
		app.ReceiptsPage(w, httptest.NewRequest(http.MethodGet, "/receipts?after="+after, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("after=%q: expected 400, got %d", after, w.Code)
		}
	}
}

// TestReceiptsList_AfterFromDifferentQuery проверяет, что курсор одной
// выборки безопасно применять к другой: WHERE (поиск и фильтры)
// применяется до курсорного предиката.
func TestReceiptsList_AfterFromDifferentQuery(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "QueryOrg", "k_query")
	app := &App{receipts: receipts.NewStore(db)}

	for _, n := range []string{"alpha-1", "beta-1", "alpha-2", "beta-2", "alpha-3", "beta-3"} {
		insertReceiptRaw(t, db, orgID, n, "2026-08-01")
	}

	// Первая порция по alpha: две самых новых (id DESC) строки alpha.
	w := httptest.NewRecorder()
	app.ReceiptsPage(w, httptest.NewRequest(http.MethodGet, "/receipts?q=alpha&limit=2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	alphaBody := w.Body.String()
	if got := len(receiptEditURLs(t, alphaBody)); got != 2 {
		t.Fatalf("expected 2 alpha rows, got %d", got)
	}
	loadMore := receiptLoadMoreURL(t, alphaBody)
	if loadMore == "" {
		t.Fatal("expected sentinel on alpha page")
	}
	after := receiptAfterFromLoadMoreURL(t, loadMore)
	if after == "" {
		t.Fatal("expected after param in load-more URL")
	}

	// Применяем курсор alpha к выборке beta: оставшиеся после позиции
	// документы обязаны быть beta.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/receipts?q=beta&limit=10&after="+after, nil)
	r2.Header.Set("HX-Request", "true")
	app.ReceiptsPage(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	betaBody := w2.Body.String()
	if strings.Contains(betaBody, "alpha") {
		t.Fatalf("alpha document leaked into beta page: %s", betaBody)
	}
	// Номер выводится с подсветкой совпадений поиска (<mark>beta</mark>-1).
	urls := receiptEditURLs(t, betaBody)
	if len(urls) != 1 || urls[0] != "/receipts/2" {
		t.Fatalf("expected single beta-1 row, got %v:\n%s", urls, betaBody)
	}
}

// TestReceiptsList_StatusCell — порядок ячеек журнала и семантический
// класс статуса. Проверяется структура целиком: Номер → Дата → Организация →
// Пользователь → Контрагент → Сумма → Статус → Действия. Статус передаётся
// как receipts-status is-<key> без inline-стиля; у .receipts-row и ячеек
// фонового цвета и цвета текста нет (цвет определяет CSS темой).
func TestReceiptsList_StatusCell(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "StatusOrg", "k1")

	insertReceiptForOrg(t, db, orgID) // Чек 1: «Создан», сверху (id больше).
	uSent := insertReceiptForOrg(t, db, orgID)

	// Чек 2 становится опубликованным → «Отправлен».
	now := time.Now().Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE receipts SET sent_at = ? WHERE uuid = ?`, now, uSent); err != nil {
		t.Fatal(err)
	}

	app := &App{receipts: receipts.NewStore(db)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	if strings.Contains(body, `<div class="receipts-row" style=`) {
		t.Fatal("receipts-row must not carry inline status style")
	}

	rows := strings.Split(body, `<div class="receipts-row">`)
	// rows[0] — header + инструменты; далее data-строки.
	if len(rows) < 3 {
		t.Fatalf("expected 2 data rows, got %d", len(rows)-1)
	}
	sentRow, createdRow := rows[1], rows[2]

	for name, block := range map[string]string{"sent": sentRow, "created": createdRow} {
		info := indexesOf(block, `<div class="receipts-cell"`)
		if len(info) != 7 {
			t.Fatalf("%s: expected 7 info cells, got %d", name, len(info))
		}
		actions := indexesOf(block, `<div class="receipts-cell is-actions">`)
		if len(actions) != 1 || actions[0] < info[6] {
			t.Fatalf("%s: actions cell must follow the 7 info cells", name)
		}
	}

	// Отправлен: семантический класс is-sent, без inline-стиля.
	if !strings.Contains(sentRow, `class="receipts-status is-sent">Отправлен`) {
		t.Fatalf("sent: expected receipts-status is-sent class:\n%s", sentRow)
	}
	// Создан: класс is-created сохраняется для единообразия модели.
	if !strings.Contains(createdRow, `class="receipts-status is-created">Создан`) {
		t.Fatalf("created: expected receipts-status is-created class:\n%s", createdRow)
	}

	// Ни у строки, ни у ячейки нет inline-цвета/фона статуса.
	if strings.Contains(sentRow, `style="`) {
		t.Fatalf("sent: inline style is not allowed on status:\n%s", sentRow)
	}
	for name, block := range map[string]string{"sent": sentRow, "created": createdRow} {
		if strings.Contains(block, "background-color") {
			t.Fatalf("%s: status must not set background-color:\n%s", name, block)
		}
		if strings.Contains(block, `style="color:`) {
			t.Fatalf("%s: status must not set inline color:\n%s", name, block)
		}
	}

	// Данные собираются в «Создан» → «Отправлен» → ... (id DESC). Первая
	// строка (с бóльшим id) — опубликованный чек со статусом «Отправлен».
	if strings.Index(body, "Отправлен") > strings.Index(body, "Создан") {
		t.Fatalf("expected Отправлен row before Создан row:\n%s", body)
	}
}

// TestReceiptsList_ActionInfo — колонка «Инфо» отображает действие всегда,
// а неподтверждённое 1С состояние обозначается классом is-pending (красный).
func TestReceiptsList_ActionInfo(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActInfoOrg", "kact")
	store := receipts.NewStore(db)
	recUUID := insertReceiptForOrg(t, db, orgID)

	doc, err := store.GetByExternal(context.Background(), recUUID)
	if err != nil {
		t.Fatal(err)
	}
	id := doc.Receipt.ID

	render := func() string {
		app := &App{receipts: receipts.NewStore(db)}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
		app.ReceiptsPage(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}

	// Действия нет — колонка «Инфо» пустая.
	if body := render(); strings.Contains(body, "receipts-info") {
		t.Fatalf("expected no action info without action:\n%s", body)
	}

	// Действие установлено, не подтверждено — красный индикатор is-pending.
	if _, err := store.SetAction(context.Background(), id, receipts.ActionDelete); err != nil {
		t.Fatal(err)
	}
	if body := render(); !strings.Contains(body, `class="receipts-info is-pending">Удалить</span>`) {
		t.Fatalf("expected pending action badge:\n%s", body)
	}

	// Подтверждено 1С — текст действия остаётся, is-pending исчезает.
	if _, err := store.ConfirmActions(context.Background(), orgID, []string{recUUID}); err != nil {
		t.Fatal(err)
	}
	body := render()
	if !strings.Contains(body, `class="receipts-info">Удалить</span>`) {
		t.Fatalf("expected confirmed action badge:\n%s", body)
	}
	if strings.Contains(body, "receipts-info is-pending") {
		t.Fatalf("expected no pending badge after confirm:\n%s", body)
	}

	// Отмена — колонка «Инфо» снова пустая (запись с пустым action остаётся).
	if _, err := store.SetAction(context.Background(), id, ""); err != nil {
		t.Fatal(err)
	}
	if body := render(); strings.Contains(body, "receipts-info") {
		t.Fatalf("expected empty info after cancel:\n%s", body)
	}
}

// indexesOf возвращает абсолютные позиции всех вхождений подстроки.
func indexesOf(s, sub string) []int {
	var res []int
	pos := 0
	for pos < len(s) {
		i := strings.Index(s[pos:], sub)
		if i < 0 {
			break
		}
		res = append(res, pos+i)
		pos += i + len(sub)
	}
	return res
}

// TestReceiptCard_FilesBlock — в карточке просмотра блок «Файлы»
// выводится только при наличии файлов, иначе отсутствует.
func TestReceiptCard_FilesBlock(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	filesDB := testutil.NewTestDB(t, NewFilesSchema())
	orgID, _ := insertOrg(t, db, "CardOrg", "k1")

	// Два отправленных чека: первый без файлов, второй с файлом.
	uEmpty := insertReceiptForOrg(t, db, orgID)
	uWith := insertReceiptForOrg(t, db, orgID)

	app := &App{
		receipts:     receipts.NewStore(db),
		receiptFiles: receipts.NewFileStore(filesDB),
	}

	docWith, _ := app.receipts.GetByExternal(context.Background(), uWith)
	if _, _, err := app.receiptFiles.Upsert(context.Background(), docWith.Receipt.ID, "f1", "карта.pdf", "application/pdf", pdfBody); err != nil {
		t.Fatal(err)
	}
	docEmpty, _ := app.receipts.GetByExternal(context.Background(), uEmpty)

	now := time.Now().Format(time.RFC3339)
	for _, doc := range []*receipts.Document{docEmpty, docWith} {
		if _, err := db.Exec(`UPDATE receipts SET sent_at = ? WHERE id = ?`, now, doc.Receipt.ID); err != nil {
			t.Fatal(err)
		}
	}

	renderCard := func(id int64) string {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/receipts/"+strconv.FormatInt(id, 10)+"?mode=view", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(id, 10))
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		app.ReceiptCard(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("render card %d: expected 200, got %d: %s", id, w.Code, w.Body.String())
		}
		return w.Body.String()
	}

	withBody := renderCard(docWith.Receipt.ID)
	if !strings.Contains(withBody, ">Файлы</h2>") || !strings.Contains(withBody, "карта.pdf") {
		t.Fatalf("expected files block with file name in card with files:\n%s", withBody)
	}

	emptyBody := renderCard(docEmpty.Receipt.ID)
	if strings.Contains(emptyBody, ">Файлы</h2>") {
		t.Fatalf("expected no files block for doc without files:\n%s", emptyBody)
	}
}

func TestReceiptReturnURL(t *testing.T) {
	doc := &receipts.Document{
		Receipt: &receipts.Receipt{ID: 42},
	}
	cases := []struct {
		from string
		want string
	}{
		{receiptFromEdit, "/receipts/42"},
		{"list", RouteReceipts},
		{"", RouteReceipts},
		{"render", RouteReceipts},
	}
	for _, c := range cases {
		if got := receiptReturnURL(doc, c.from); got != c.want {
			t.Errorf("receiptReturnURL(from=%q) = %q, want %q", c.from, got, c.want)
		}
	}
}

// setupSendableReceipt создаёт App и возвращает unsent документ в списке,
// готовый к отправке. App включает receipts, organizations и products.
func setupSendableReceipt(t *testing.T) (*App, *receipts.Document) {
	t.Helper()
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Org", "key")
	prodStore := products.NewStore(db)
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}
	prodID, _ := insertProduct(t, db, orgID, "Send Product", "pcs")
	body := "number=009&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=1&customer_id=1&total=100&date=2026-07-29" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][unit]=pcs&items[0][quantity]=1&items[0][price]=100&items[0][amount]=100"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("setup: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("setup: expected one receipt, got %d: %v", len(list), err)
	}
	doc, err := app.receipts.GetByID(context.Background(), list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return app, doc
}

func TestReceiptSendConfirmPage_FromEditCancel(t *testing.T) {
	app, doc := setupSendableReceipt(t)

	idStr := strconv.FormatInt(doc.Receipt.ID, 10)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+idStr+"?mode=send&from=edit", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "data-confirm=") {
		t.Fatalf("expected send confirmation form, got %s", body)
	}
	if !strings.Contains(body, `method="POST" action="/receipts/`+idStr+`/send"`) {
		t.Fatalf("expected send form to POST /receipts/%s/send, got %s", idStr, body)
	}
	if !strings.Contains(body, `href="/receipts/`+idStr+`"`) {
		t.Fatalf("expected cancel href to editor (from=edit), got %s", body)
	}
}

func TestReceiptSendConfirmPage_FromListCancel(t *testing.T) {
	app, doc := setupSendableReceipt(t)

	idStr := strconv.FormatInt(doc.Receipt.ID, 10)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+idStr+"?mode=send&from=list", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)

	body := w.Body.String()
	if !strings.Contains(body, `href="/receipts"`) {
		t.Fatalf("expected cancel href to list for from=list, got %s", body)
	}
}

func TestReceiptSendConfirmPage_NoButtonsAfterSend(t *testing.T) {
	app, doc := setupSendableReceipt(t)

	// Отправим документ напрямую в хранилище.
	now := time.Now()
	doc.Receipt.SentAt = &now
	if err := app.receipts.Save(context.Background(), doc); err != nil {
		t.Fatal(err)
	}

	idStr := strconv.FormatInt(doc.Receipt.ID, 10)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+idStr+"?mode=send", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)

	body := w.Body.String()
	if strings.Contains(body, "data-confirm=") {
		t.Errorf("expected no confirmation form for already sent document, got %s", body)
	}
	if strings.Contains(body, "Отмена") {
		t.Errorf("expected no cancel button for already sent document, got %s", body)
	}
}

// getReceiptCard выполняет GET /receipts/{id}[?mode=...] через ReceiptCard.
func getReceiptCard(t *testing.T, app *App, idStr, mode string) *httptest.ResponseRecorder {
	t.Helper()
	target := "/receipts/" + idStr
	if mode != "" {
		target += "?mode=" + mode
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)
	return w
}

// TestReceiptCard_ViewMode: несохранённый чек в mode=view — read-only просмотр.
// Карточка рендерится через ReceiptCardPage, а не ReceiptSendConfirmPage,
// поэтому не должно быть ни confirm-формы, ни формы отправки.
func TestReceiptCard_ViewMode(t *testing.T) {
	app, doc := setupSendableReceipt(t)

	idStr := strconv.FormatInt(doc.Receipt.ID, 10)
	w := getReceiptCard(t, app, idStr, "view")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Товарный чек") {
		t.Errorf("expected card content, got %s", body)
	}
	if strings.Contains(body, "data-confirm=") {
		t.Errorf("expected no send confirmation form in view mode, got %s", body)
	}
	if strings.Contains(body, `action="/receipts/`+idStr+`/send"`) {
		t.Errorf("expected no send form action in view mode, got %s", body)
	}
}

// TestReceiptCard_ReadOnlySent: отправленный чек без mode — штатный просмотр.
// canEdit=false → read-only ветка через ReceiptCardPage: без действий отправки.
func TestReceiptCard_ReadOnlySent(t *testing.T) {
	app, doc := setupSendableReceipt(t)

	now := time.Now()
	doc.Receipt.SentAt = &now
	if err := app.receipts.Save(context.Background(), doc); err != nil {
		t.Fatal(err)
	}

	idStr := strconv.FormatInt(doc.Receipt.ID, 10)
	w := getReceiptCard(t, app, idStr, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Товарный чек") {
		t.Errorf("expected card content, got %s", body)
	}
	if strings.Contains(body, "data-confirm=") {
		t.Errorf("expected no send confirmation form for sent document, got %s", body)
	}
	if strings.Contains(body, `action="/receipts/`+idStr+`/send"`) {
		t.Errorf("expected no send form action for sent document, got %s", body)
	}
}

func TestReceiptSubmit_SuccessRedirectsToFlash(t *testing.T) {
	app, doc := setupSendableReceipt(t)

	idStr := strconv.FormatInt(doc.Receipt.ID, 10)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts/"+idStr+"/send", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptSubmit(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != RouteReceipts {
		t.Fatalf("expected redirect to %s, got %s", RouteReceipts, loc)
	}
	got, err := app.receipts.GetByID(context.Background(), doc.Receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Receipt.SentAt == nil {
		t.Fatal("expected SentAt to be set after successful send")
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

// saveCustomer создаёт контрагента в организации и возвращает его ID.
func saveCustomer(t *testing.T, db *sql.DB, orgID int64, name string) int64 {
	t.Helper()
	store := customers.NewStore(db)
	c := store.New()
	c.UUID = "uuid-" + name
	c.Name = name
	c.OrganizationID = orgID
	if err := store.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	return c.ID
}

// TestReceiptsList_FilterParams — расширенный отбор применён: список
// сужается по периоду, сумме, организации, контрагенту и статусу;
// панель открыта, кнопка «Фильтр» активна, значения попадают в payload.
func TestReceiptsList_FilterParams(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "FilterOrg", "kf")
	custID := saveCustomer(t, db, orgID, "Filter Customer")

	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		customers:     customers.NewStore(db),
	}

	rec := &receipts.Receipt{
		Number:         "F001",
		Date:           time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     custID,
		Total:          777,
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	params := url.Values{}
	params.Set("date_from", "2026-05-01")
	params.Set("date_to", "2026-05-31")
	params.Set("amount_from", "700")
	params.Set("amount_to", "800")
	params.Set("organization_id", strconv.FormatInt(orgID, 10))
	params.Set("customer_id", strconv.FormatInt(custID, 10))
	params.Set("status", receipts.StatusCreated)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts?"+params.Encode(), nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	if !strings.Contains(body, "F001") {
		t.Fatalf("expected receipt in filtered list:\n%s", body)
	}
	if !strings.Contains(body, `class="button filter-toggle is-info"`) {
		t.Fatal("expected filter toggle button to be active")
	}
	if !strings.Contains(body, `&#34;open&#34;:true`) {
		t.Fatal("expected filter panel payload to be open")
	}
	if !strings.Contains(body, `class="filter-panel"`) {
		t.Fatal("expected filter panel markup")
	}
	if !strings.Contains(body, `<option value="`+receipts.StatusCreated+`" selected>`+receipts.StatusCreated) {
		t.Fatal("expected created status option selected")
	}
	// Серверное имя выбранного контрагента попадает в payload справочника.
	if !strings.Contains(body, "Filter Customer") {
		t.Fatal("expected customer dictionary in filter payload")
	}
	// Выбранный контрагент попадает в payload.
	if !strings.Contains(body, "&#34;customerId&#34;:"+strconv.FormatInt(custID, 10)) {
		t.Fatal("expected selected customerId in filter payload")
	}
}

// TestReceiptsList_FilterCustomerPicker — контрагент в панели фильтра
// выбирается поисковым picker (модальное окно), а не <select>: панель
// содержит поле отображения, кнопки «Выбрать»/«Очистить», скрытый input
// customer_id и модалку с поиском и списком.
func TestReceiptsList_FilterCustomerPicker(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "FilterOrg", "kf")
	saveCustomer(t, db, orgID, "Иванов ООО")
	saveCustomer(t, db, orgID, "Альфа ООО")

	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		customers:     customers.NewStore(db),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	// Справочник контрагентов приходит целиком в payload.
	if !strings.Contains(body, "Иванов ООО") || !strings.Contains(body, "Альфа ООО") {
		t.Fatal("expected customers dictionary in filter payload")
	}

	// UI: поисковый picker вместо <select>.
	if strings.Contains(body, `<select name="customer_id"`) {
		t.Fatal("expected customer picker to replace <select>")
	}
	if !strings.Contains(body, `name="customer_id"`) {
		t.Fatal("expected hidden customer_id input")
	}
	if !strings.Contains(body, `@click="openCustomerPicker()"`) {
		t.Fatal("expected customer picker open button")
	}
	if !strings.Contains(body, `@click="clearCustomer()"`) {
		t.Fatal("expected customer clear button")
	}
	if !strings.Contains(body, "Выбор контрагента") {
		t.Fatal("expected customer picker modal")
	}
	if !strings.Contains(body, `placeholder="Поиск контрагентов..."`) {
		t.Fatal("expected search field in customer picker modal")
	}
}

// TestReceiptsList_FilterExcludes — чек вне заданного периода отсеивается.
func TestReceiptsList_FilterExcludes(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "FilterOrg", "kf")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	rec := &receipts.Receipt{
		Number:         "F002",
		Date:           time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          50,
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts?date_from=2026-06-01", nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "F002") {
		t.Fatal("expected receipt to be excluded by date filter")
	}
}

// TestReceiptsList_FilterPairValidation — контрагент, принадлежащий другой
// организации, чем выбранная, отбрасывается на сервере (не только в UI):
// фильтр по контрагенту не применяется, список не сужается.
func TestReceiptsList_FilterPairValidation(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "OrgA", "ka")
	orgBID, _ := insertOrg(t, db, "OrgB", "kb")
	foreignCust := saveCustomer(t, db, orgBID, "Foreign Customer")

	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		customers:     customers.NewStore(db),
	}

	// Чек в OrgA.
	rec := &receipts.Receipt{
		Number:         "F003",
		Date:           time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          100,
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	// customer_id принадлежит OrgB, но organization_id = OrgA — пара
	// невалидна, контрагент должен быть сброшен и фильтр не сузить список.
	params := url.Values{}
	params.Set("organization_id", strconv.FormatInt(orgID, 10))
	params.Set("customer_id", strconv.FormatInt(foreignCust, 10))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts?"+params.Encode(), nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "F003") {
		t.Fatalf("expected receipt to remain after dropping invalid customer:\n%s", body)
	}
	// В payload customerId сброшен в 0.
	if !strings.Contains(body, `&#34;customerId&#34;:0`) {
		t.Fatal("expected invalid customer to be dropped from filter payload")
	}
}

// TestReceiptsList_FragmentRendersBrowser — фрагмент-ответ (живой поиск /
// применение фильтра) перерисовывает обёртку receipts-browser целиком.
func TestReceiptsList_FragmentRendersBrowser(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	insertOrg(t, db, "FilterOrg", "kf")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	r.Header.Set("HX-Request", "true")
	app.ReceiptsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `<div id="receipts-browser"`) {
		t.Fatal("expected fragment to render the receipts-browser wrapper")
	}
	if strings.Contains(body, "/static/favicon.ico") {
		t.Fatal("expected fragment without the base layout")
	}
}

// TestReceiptsList_MarkMatches — найденные слова подсвечиваются тегом
// <mark> в ячейках текстового поиска.
func TestReceiptsList_MarkMatches(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "Ромашка", "kf")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
	}

	rec := &receipts.Receipt{
		Number:         "F004",
		Date:           time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local),
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          100,
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts?q=ромашка", nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "<mark>Ромашка</mark>") {
		t.Fatalf("expected highlighted match in organization cell:\n%s", body)
	}
}

// TestParseReceiptFilter — разбор параметров расширенного отбора:
// невалидные значения отбрасываются, валидные проходят.
func TestParseReceiptFilter(t *testing.T) {
	cases := []struct {
		name   string
		params map[string]string
		check  func(t *testing.T, f receiptFilter)
	}{
		{
			name: "valid all",
			params: map[string]string{
				"date_from":       "2026-05-01",
				"date_to":         "2026-05-31",
				"amount_from":     "100.5",
				"amount_to":       "900",
				"organization_id": "7",
				"customer_id":     "9",
				"status":          receipts.StatusAccepted,
			},
			check: func(t *testing.T, f receiptFilter) {
				if f.dateFrom != "2026-05-01" || f.dateTo != "2026-05-31" {
					t.Error("expected dates parsed")
				}
				if f.amountFrom == nil || *f.amountFrom != 100.5 || f.amountTo == nil || *f.amountTo != 900 {
					t.Error("expected amounts parsed")
				}
				if f.orgID != 7 || f.custID != 9 || f.status != receipts.StatusAccepted {
					t.Error("expected ids and status parsed")
				}
				if !f.active() {
					t.Error("expected filter active")
				}
			},
		},
		{
			name: "invalid dropped",
			params: map[string]string{
				"date_from":   "01.05.2026",
				"amount_from": "abc",
				"amount_to":   "-5",
				"org":         "x",
			},
			check: func(t *testing.T, f receiptFilter) {
				if f.dateFrom != "" {
					t.Error("expected invalid date dropped")
				}
				if f.amountFrom != nil || f.amountTo != nil {
					t.Error("expected invalid amounts dropped")
				}
				if f.orgID != 0 || f.custID != 0 {
					t.Error("expected invalid ids dropped")
				}
				if f.active() {
					t.Error("expected inactive filter for all-invalid params")
				}
			},
		},
		{
			name:   "unknown status dropped",
			params: map[string]string{"status": "Неизвестный"},
			check: func(t *testing.T, f receiptFilter) {
				if f.status != "" {
					t.Error("expected unknown status dropped")
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target := "/receipts"
			first := true
			for k, v := range c.params {
				if first {
					target += "?"
					first = false
				} else {
					target += "&"
				}
				target += url.QueryEscape(k) + "=" + url.QueryEscape(v)
			}
			r := httptest.NewRequest(http.MethodGet, target, nil)
			c.check(t, parseReceiptFilter(r))
		})
	}
}

// saveAppReceipt создаёт чек напрямую через Store с заданными полями.
func saveAppReceipt(t *testing.T, app *App, orgID int64, number, status string, sent bool) *receipts.Receipt {
	t.Helper()
	now := time.Now()
	rec := &receipts.Receipt{
		Number:         number,
		Date:           now,
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          10,
		Status:         status,
	}
	if sent {
		rec.SentAt = &now
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}
	return rec
}

// markUserRequest возвращает POST-запрос пометки с установленным
// URL-параметром id и идентичностью пользователя в контексте.
func markUserRequest(t *testing.T, id string, u users.Identity) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/receipts/"+id+"/delete", nil)
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, u))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestReceiptMarkDeleted_RequiresAdmin(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "DelOrg", "kdel")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveAppReceipt(t, app, orgID, "MDROLE001", receipts.StatusCreated, false)
	idStr := strconv.FormatInt(rec.ID, 10)

	post := func(u users.Identity) *httptest.ResponseRecorder {
		h := RequireAdmin(http.HandlerFunc(app.ReceiptMarkDeleted))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, markUserRequest(t, idStr, u))
		return w
	}

	if w := post(users.Identity{}); w.Code != http.StatusForbidden {
		t.Fatalf("no user: expected 403, got %d", w.Code)
	}
	if w := post(users.Identity{ID: 1, Login: "op", IsAdmin: false}); w.Code != http.StatusForbidden {
		t.Fatalf("non-admin: expected 403, got %d", w.Code)
	}

	w := post(users.Identity{ID: 2, Login: "admin", IsAdmin: true})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("admin: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != RouteReceipts {
		t.Fatalf("admin: expected redirect to %s, got %s", RouteReceipts, loc)
	}
	if _, err := app.receipts.GetByID(context.Background(), rec.ID); err != receipts.ErrNotFound {
		t.Fatalf("admin: marked document should be hidden, got %v", err)
	}
}

func TestReceiptMarkDeleted_StatusForbidden(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "DelOrg2", "kdel2")
	app := &App{receipts: receipts.NewStore(db)}
	admin := users.Identity{ID: 2, Login: "admin", IsAdmin: true}

	for _, status := range []string{receipts.StatusAccepted, receipts.StatusProcessed, receipts.StatusFinished} {
		rec := saveAppReceipt(t, app, orgID, "MDNF-"+status, status, true)
		idStr := strconv.FormatInt(rec.ID, 10)

		h := RequireAdmin(http.HandlerFunc(app.ReceiptMarkDeleted))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, markUserRequest(t, idStr, admin))

		if w.Code != http.StatusForbidden {
			t.Fatalf("status %q: expected 403, got %d", status, w.Code)
		}
		if _, err := app.receipts.GetByID(context.Background(), rec.ID); err != nil {
			t.Fatalf("status %q: document should remain active, got %v", status, err)
		}
	}
}

func TestReceiptMarkDeleted_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "DelOrg3", "kdel3")
	app := &App{receipts: receipts.NewStore(db)}
	admin := users.Identity{ID: 2, Login: "admin", IsAdmin: true}

	post := func(id string) *httptest.ResponseRecorder {
		h := RequireAdmin(http.HandlerFunc(app.ReceiptMarkDeleted))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, markUserRequest(t, id, admin))
		return w
	}

	if w := post("999999"); w.Code != http.StatusNotFound {
		t.Fatalf("unknown id: expected 404, got %d", w.Code)
	}

	rec := saveAppReceipt(t, app, orgID, "MDNOT", receipts.StatusCreated, false)
	idStr := strconv.FormatInt(rec.ID, 10)
	if w := post(idStr); w.Code != http.StatusSeeOther {
		t.Fatalf("first mark: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	if w := post(idStr); w.Code != http.StatusNotFound {
		t.Fatalf("repeat mark: expected 404, got %d", w.Code)
	}
}

func TestReceiptsPage_DeleteButton_RoleAndStatus(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "DelOrg4", "kdel4")
	app := &App{receipts: receipts.NewStore(db)}

	created := saveAppReceipt(t, app, orgID, "MDUIP1", receipts.StatusCreated, false)
	accepted := saveAppReceipt(t, app, orgID, "MDUIP2", receipts.StatusAccepted, true)

	get := func(u users.Identity) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := markUserRequest(t, "", u)
		r = httptest.NewRequest(http.MethodGet, RouteReceipts, nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, u))
		app.ReceiptsPage(w, r)
		return w
	}

	bodyAdmin := get(users.Identity{ID: 2, Login: "admin", IsAdmin: true}).Body.String()
	createdID := strconv.FormatInt(created.ID, 10)
	acceptedID := strconv.FormatInt(accepted.ID, 10)
	if !strings.Contains(bodyAdmin, `method="POST" action="/receipts/`+createdID+`/delete"`) {
		t.Error("expected delete form for active receipt for admin")
	}
	if !strings.Contains(bodyAdmin, "Пометить товарный чек №"+created.Number+" на удаление?") {
		t.Error("expected data-confirm text with receipt number")
	}
	if strings.Contains(bodyAdmin, `action="/receipts/`+acceptedID+`/delete"`) {
		t.Error("expected no delete form for non-deletable status")
	}

	bodyNonAdmin := get(users.Identity{ID: 1, Login: "op", IsAdmin: false}).Body.String()
	if strings.Contains(bodyNonAdmin, "/delete") {
		t.Error("expected no delete form for non-admin")
	}
}

func TestReceiptSave_RoundsToTwoDecimals(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "RoundOrg", "kround")
	prodStore := products.NewStore(db)
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      prodStore,
	}
	prod1, _ := insertProduct(t, db, orgID, "Product R1", "pcs")
	prod2, _ := insertProduct(t, db, orgID, "Product R2", "pcs")

	// Строка i: {product, quantity, price, amount}. amount — доверенное
	// значение; пустое amount даёт legacy-фолбэк round2(qty*price).
	line := func(i int64, product int64, quantity, price, value string, valueSet bool) string {
		parts := "items[" + strconv.FormatInt(i, 10) + "][product_id]=" + strconv.FormatInt(product, 10) +
			"&items[" + strconv.FormatInt(i, 10) + "][quantity]=" + quantity +
			"&items[" + strconv.FormatInt(i, 10) + "][price]=" + price
		if valueSet {
			parts += "&items[" + strconv.FormatInt(i, 10) + "][amount]=" + value
		}
		return parts
	}

	var body strings.Builder
	body.WriteString("number=900&organization_id=" + strconv.FormatInt(orgID, 10))
	body.WriteString("&user_id=1&customer_id=1&date=2026-07-30")
	body.WriteString("&" + line(0, prod1, "1.999", "1", "", false))
	body.WriteString("&" + line(1, prod1, "3", "10", "10.00", true))
	body.WriteString("&" + line(2, prod2, "2", "5", "7.00", true))
	body.WriteString("&" + line(3, prod2, "1.2345", "2.3456", "3.4567", true))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body.String()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ReceiptSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}

	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected one receipt, got %d: %v", len(list), err)
	}
	doc, err := app.receipts.GetByID(context.Background(), list[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(doc.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(doc.Items))
	}

	// Строка 0: legacy-пустое amount → round2(qty × price). Количество
	// сохраняет 3 знака (round3), цена и сумма — 2 знака.
	q0, p0, a0 := doc.Items[0].Quantity, doc.Items[0].Price, doc.Items[0].Amount
	if q0 != 1.999 || p0 != 1.00 || a0 != 2.00 {
		t.Fatalf("line 0: got qty=%v price=%v amount=%v, want 1.999/1/2", q0, p0, a0)
	}
	// Строка 1: amount=10.00 доверенное — НЕ заменяется qty*price (30).
	if doc.Items[1].Amount != 10.00 {
		t.Fatalf("line 1: trusted amount should stay 10.00, got %v", doc.Items[1].Amount)
	}
	// Строка 2: amount="7.00" — доверенное, не заменяется qty*price (10).
	if doc.Items[2].Amount != 7.00 {
		t.Fatalf("line 2: trusted amount should stay 7.00, got %v", doc.Items[2].Amount)
	}
	// Строка 3: количество round3(1.2345)=1.235; цена round2(2.3456)=2.35;
	// amount — доверенное round2(3.4567)=3.46.
	if q3, p3 := doc.Items[3].Quantity, doc.Items[3].Price; q3 != 1.235 || p3 != 2.35 {
		t.Fatalf("line 3: got qty=%v price=%v, want 1.235/2.35", q3, p3)
	}
	if doc.Items[3].Amount != 3.46 {
		t.Fatalf("line 3: normalized amount = %v, want 3.46", doc.Items[3].Amount)
	}

	// total = сумма округлённых amount строк: 2 + 10 + 7 + 3.46.
	if doc.Receipt.Total != 22.46 {
		t.Fatalf("total = %v, want 22.46", doc.Receipt.Total)
	}
}

// saveSyncedAppReceipt создаёт чек с назначенным внешним UUID (т.е.
// синхронизированный в 1С) и возвращает его.
func saveSyncedAppReceipt(t *testing.T, app *App, orgID int64, number string) *receipts.Receipt {
	t.Helper()
	rec := saveAppReceipt(t, app, orgID, number, receipts.StatusSent, true)
	uuid := "act-" + uniqueSuffix()
	if _, err := app.receipts.SynchronizeByID(context.Background(), orgID, []receipts.SyncUpdate{{ID: rec.ID, UUID: &uuid}}); err != nil {
		t.Fatal(err)
	}
	rec.UUID = uuid
	return rec
}

// actionRequest возвращает GET/POST-запрос модалки/сохранения действия с
// URL-параметром id.
func actionRequest(t *testing.T, method, id string, formAction string) *http.Request {
	t.Helper()
	path := "/receipts/" + id + "/action"
	r := httptest.NewRequest(method, path, nil)
	if method == http.MethodPost {
		r = httptest.NewRequest(method, path, strings.NewReader("action="+url.QueryEscape(formAction)))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestReceiptActionSave_SetAndClear(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg", "kact")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveSyncedAppReceipt(t, app, orgID, "ACT1001")
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	app.ReceiptActionSave(w, actionRequest(t, http.MethodPost, idStr, receipts.ActionDelete))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	action, receivedAt, err := app.receipts.GetAction(context.Background(), rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if action != receipts.ActionDelete || receivedAt != nil {
		t.Fatalf("set: got action %q / receivedAt %v", action, receivedAt)
	}

	w = httptest.NewRecorder()
	app.ReceiptActionSave(w, actionRequest(t, http.MethodPost, idStr, ""))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("clear: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	action, receivedAt, err = app.receipts.GetAction(context.Background(), rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if action != "" || receivedAt != nil {
		t.Fatalf("clear: got action %q / receivedAt %v", action, receivedAt)
	}
}

func TestReceiptActionSave_NotSynced(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg2", "kact2")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveAppReceipt(t, app, orgID, "ACT1002", receipts.StatusCreated, false)
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	app.ReceiptActionSave(w, actionRequest(t, http.MethodPost, idStr, receipts.ActionDelete))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-synced, got %d", w.Code)
	}
}

func TestReceiptActionSave_InvalidAction(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg3", "kact3")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveSyncedAppReceipt(t, app, orgID, "ACT1003")
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	app.ReceiptActionSave(w, actionRequest(t, http.MethodPost, idStr, "Взорвать"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid action, got %d", w.Code)
	}
}

func TestReceiptActionDialog(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg4", "kact4")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveSyncedAppReceipt(t, app, orgID, "ACT1004")
	idStr := strconv.FormatInt(rec.ID, 10)

	if _, err := app.receipts.SetAction(context.Background(), rec.ID, receipts.ActionChange); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	app.ReceiptActionDialog(w, actionRequest(t, http.MethodGet, idStr, ""))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Действие для чека № "+rec.Number) {
		t.Errorf("expected dialog title with number, got: %s", body)
	}
	if !strings.Contains(body, "Текущее действие: <strong>"+receipts.ActionChange+"</strong>") {
		t.Errorf("expected current action, got: %s", body)
	}
}

func TestReceiptActionDialog_NotSynced(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg5", "kact5")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveAppReceipt(t, app, orgID, "ACT1005", receipts.StatusCreated, false)
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	app.ReceiptActionDialog(w, actionRequest(t, http.MethodGet, idStr, ""))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-synced, got %d", w.Code)
	}
}

// cancelSyncedReceipt переводит синхронизированный чек в статус «Отменен».
func cancelSyncedReceipt(t *testing.T, app *App, orgID int64, uuid string) {
	t.Helper()
	status := receipts.StatusCancelled
	if _, err := app.receipts.UpdateByExternal(context.Background(), orgID, uuid, &status); err != nil {
		t.Fatal(err)
	}
}

func TestReceiptActionSave_Cancelled(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg6", "kact6")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveSyncedAppReceipt(t, app, orgID, "ACT1006")
	cancelSyncedReceipt(t, app, orgID, rec.UUID)
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	app.ReceiptActionSave(w, actionRequest(t, http.MethodPost, idStr, receipts.ActionDelete))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cancelled document, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceiptActionDialog_Cancelled(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg7", "kact7")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveSyncedAppReceipt(t, app, orgID, "ACT1007")
	cancelSyncedReceipt(t, app, orgID, rec.UUID)
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	app.ReceiptActionDialog(w, actionRequest(t, http.MethodGet, idStr, ""))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cancelled document, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceiptsList_ActionButtonCancelled(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "ActOrg8", "kact8")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveSyncedAppReceipt(t, app, orgID, "ACT1008")
	cancelSyncedReceipt(t, app, orgID, rec.UUID)
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); strings.Contains(body, `data-dialog-url="/receipts/`+idStr+`/action"`) {
		t.Fatalf("expected no action button for cancelled document:\n%s", body)
	}
}

// insertTestUser создаёт пользователя с логином и возвращает его ID.
func insertTestUser(t *testing.T, dbt *sql.DB, login string) int64 {
	t.Helper()
	res, err := dbt.Exec(`
		INSERT INTO users (uuid, login, email, password_hash, is_admin, created_at, updated_at)
		VALUES (?, ?, '', '', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, "user-"+login, login)
	if err != nil {
		t.Fatalf("insert user %s: %v", login, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// withUserIdentity кладёт пользователя сессии в контекст запроса.
func withUserIdentity(r *http.Request, u users.Identity) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userContextKey, u))
}

// TestReceiptSave_NewDocumentIgnoresClientDateAndUser проверяет, что при
// создании нового документа date и user_id из POST игнорируются: дата и
// автор назначаются сервером.
func TestReceiptSave_NewDocumentIgnoresClientDateAndUser(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "AuditOrg", "kaudit")
	prodID, _ := insertProduct(t, db, orgID, "Audit Product", "pcs")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      products.NewStore(db),
	}
	sessionUser := users.Identity{ID: 7, Login: "session-user"}

	body := "number=001&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=999&customer_id=1&total=1000&date=2020-01-01" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=2&items[0][price]=500&items[0][amount]=1000"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = withUserIdentity(r, sessionUser)
	app.ReceiptSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected one receipt, got %d: %v", len(list), err)
	}
	got := list[0]
	today := time.Now().Format("2006-01-02")
	if got.Date.Format("2006-01-02") != today {
		t.Fatalf("expected server date %s, got %s", today, got.Date.Format("2006-01-02"))
	}
	if got.UserID != sessionUser.ID {
		t.Fatalf("expected session user %d, got %d", sessionUser.ID, got.UserID)
	}
	if got.CreatedAt.Format("2006-01-02") != today {
		t.Fatalf("expected created_at today, got %s", got.CreatedAt.Format("2006-01-02"))
	}
}

// TestReceiptCopy_UsesServerDateAndSessionUser проверяет, что копирование
// не переносит дату и автора исходного документа: новый документ получает
// текущую дату и текущего пользователя сессии.
func TestReceiptCopy_UsesServerDateAndSessionUser(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "CopyAuditOrg", "kcopyaudit")
	prodID, _ := insertProduct(t, db, orgID, "Copy Product", "pcs")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      products.NewStore(db),
	}
	srcUserID := insertTestUser(t, db, "src-user")
	src := &receipts.Receipt{
		Number:         "SRC001",
		Date:           time.Date(2020, 1, 1, 12, 0, 0, 0, time.Local),
		OrganizationID: orgID,
		UserID:         srcUserID,
		CustomerID:     1,
		Total:          0,
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: src}); err != nil {
		t.Fatal(err)
	}
	idStr := strconv.FormatInt(src.ID, 10)
	sessionUser := users.Identity{ID: 7, Login: "session-user"}

	// GET /receipts/{id}/copy — форма нового документа.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+idStr+"/copy", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = withUserIdentity(r, sessionUser)
	app.ReceiptCopyPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("copy page: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	pageBody := w.Body.String()
	if !strings.Contains(pageBody, "Создан на основании документа №SRC001") {
		t.Fatal("expected copy banner")
	}
	if !strings.Contains(pageBody, time.Now().Format("2006-01-02")) {
		t.Fatal("expected current date in copy form")
	}
	if strings.Contains(pageBody, "src-user") {
		t.Fatal("source user must not leak into copy page")
	}

	// POST /receipts — сохранение копии с подменёнными date/user_id.
	body := "number=&organization_id=" + strconv.FormatInt(orgID, 10) +
		"&user_id=999&customer_id=1&total=1000&date=2020-01-01" +
		"&items[0][product_id]=" + strconv.FormatInt(prodID, 10) +
		"&items[0][quantity]=2&items[0][price]=500&items[0][amount]=1000"
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/receipts", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = withUserIdentity(r, sessionUser)
	app.ReceiptSave(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("save copy: expected 303, got %d: %s", w.Code, w.Body.String())
	}

	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected two receipts, got %d: %v", len(list), err)
	}
	var copied *receipts.Receipt
	for _, rec := range list {
		if rec.ID != src.ID {
			copied = rec
		}
	}
	if copied == nil {
		t.Fatal("expected a new copied receipt")
	}
	today := time.Now().Format("2006-01-02")
	if copied.Date.Format("2006-01-02") != today {
		t.Fatalf("copied: expected server date %s, got %s", today, copied.Date.Format("2006-01-02"))
	}
	if copied.UserID != sessionUser.ID {
		t.Fatalf("copied: expected session user %d, got %d", sessionUser.ID, copied.UserID)
	}

	srcAfter, err := app.receipts.GetByID(context.Background(), src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if srcAfter.Receipt.Date.Format("2006-01-02") != "2020-01-01" || srcAfter.Receipt.UserID != srcUserID {
		t.Fatalf("source receipt changed: date=%s user=%d",
			srcAfter.Receipt.Date.Format("2006-01-02"), srcAfter.Receipt.UserID)
	}
}

// TestReceiptCard_ViewShowsDateWithTime проверяет, что read-only карточка
// показывает дату со временем создания.
func TestReceiptCard_ViewShowsDateWithTime(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "DTOrg", "kdt")
	app := &App{receipts: receipts.NewStore(db)}
	rec := saveAppReceipt(t, app, orgID, "DT001", receipts.StatusCreated, false)
	idStr := strconv.FormatInt(rec.ID, 10)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+idStr+"?mode=view", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.ReceiptCard(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	want := rec.CreatedAt.Format("02.01.2006 15:04")
	if !strings.Contains(w.Body.String(), want) {
		t.Fatalf("expected date with time %q in card:\n%s", want, w.Body.String())
	}
}

// TestReceiptCard_ViewShowsSentAt проверяет, что read-only карточка
// показывает дату отправки документа в 1С, и скрывает её до отправки.
func TestReceiptCard_ViewShowsSentAt(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "SentAtOrg", "ksentat")
	app := &App{receipts: receipts.NewStore(db)}

	render := func(rec *receipts.Receipt) string {
		t.Helper()
		idStr := strconv.FormatInt(rec.ID, 10)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/receipts/"+idStr+"?mode=view", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", idStr)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		app.ReceiptCard(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}

	sent := saveAppReceipt(t, app, orgID, "SENT001", receipts.StatusSent, true)
	body := render(sent)
	if !strings.Contains(body, "Отправлен в 1С") {
		t.Fatalf("expected send label in card:\n%s", body)
	}
	want := sent.SentAt.Format("02.01.2006 15:04")
	if !strings.Contains(body, want) {
		t.Fatalf("expected send date %q in card:\n%s", want, body)
	}

	unsent := saveAppReceipt(t, app, orgID, "SENT002", receipts.StatusCreated, false)
	if body := render(unsent); strings.Contains(body, "Отправлен в 1С") {
		t.Fatalf("expected no send row before sending:\n%s", body)
	}
}

// TestReceiptsList_ShowsUserColumn проверяет колонку «Пользователь».
func TestReceiptsList_ShowsUserColumn(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "UserColOrg", "kuc")
	userID := insertTestUser(t, db, "creator-login")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		users:         users.NewStore(db),
	}
	rec := &receipts.Receipt{
		Number:         "UC001",
		Date:           time.Now(),
		OrganizationID: orgID,
		UserID:         userID,
		CustomerID:     1,
		Total:          10,
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	app.ReceiptsPage(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "creator-login") {
		t.Fatal("expected creator login in list")
	}
}

// TestReceiptsList_FilterByUser проверяет отбор по пользователю и
// отбрасывание неизвестного user_id.
func TestReceiptsList_FilterByUser(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "FilterUserOrg", "kfu")
	userA := insertTestUser(t, db, "filter-user-a")
	userB := insertTestUser(t, db, "filter-user-b")
	app := &App{
		receipts:      receipts.NewStore(db),
		organizations: organizations.NewStore(db),
		users:         users.NewStore(db),
	}
	save := func(number string, userID int64) {
		rec := &receipts.Receipt{
			Number:         number,
			Date:           time.Now(),
			OrganizationID: orgID,
			UserID:         userID,
			CustomerID:     1,
			Total:          10,
		}
		if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
			t.Fatal(err)
		}
	}
	save("FU001", userA)
	save("FU002", userB)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts?user_id="+strconv.FormatInt(userA, 10), nil)
	app.ReceiptsPage(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "FU001") {
		t.Fatalf("expected user A receipt:\n%s", body)
	}
	if strings.Contains(body, "FU002") {
		t.Fatalf("did not expect user B receipt:\n%s", body)
	}

	// Неизвестный user_id отбрасывается — список не сужается.
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/receipts?user_id=999999", nil)
	app.ReceiptsPage(w, r)
	body = w.Body.String()
	if !strings.Contains(body, "FU001") || !strings.Contains(body, "FU002") {
		t.Fatalf("unknown user filter must be dropped:\n%s", body)
	}
}
