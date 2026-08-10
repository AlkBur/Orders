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
	"time"

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
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected one saved receipt, got %d: %v", len(list), err)
	}
	redirect := w.Header().Get("HX-Redirect")
	want := list[0].URL() + "?mode=send&from=edit"
	if redirect != want {
		t.Errorf("expected HX-Redirect %q, got %q", want, redirect)
	}
	if doc, err := app.receipts.GetByID(context.Background(), list[0].ID); err == nil && doc.Receipt.SentAt != nil {
		t.Error("expected SentAt to stay nil after editor send")
	}
}

func TestReceiptSend_FullPageEmptyItemsRedirectsToConfirm(t *testing.T) {
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

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}
	list, err := app.receipts.List(context.Background(), receipts.ListOptions{}, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected one saved receipt, got %d: %v", len(list), err)
	}
	want := list[0].URL() + "?mode=send&from=edit"
	if got := w.Header().Get("Location"); got != want {
		t.Errorf("expected Location %q, got %q", want, got)
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

// TestReceiptsList_StatusCell — порядок ячеек журнала и расположение цвета
// статуса. Проверяется структура целиком: Номер → Дата → Организация →
// Контрагент → Сумма → Статус → Действия. Цвет присутствует только на
// 6-й ячейке (статус); у .receipts-row inline-стиля статуса нет.
func TestReceiptsList_StatusCell(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "StatusOrg", "k1")

	insertReceiptForOrg(t, db, orgID) // Чек 1: «Создан», сверху (id больше).
	uSent := insertReceiptForOrg(t, db, orgID)

	// Чек 2 становится опубликованным → «Отправлен», #00BFFF.
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

	// Данные собираются в «Создан» → «Отправлен» → ... (id DESC). Первая
	// строка (с бóльшим id) — опубликованный чек со статусом «Отправлен».
	if strings.Index(body, "Отправлен") > strings.Index(body, "Создан") {
		t.Fatalf("expected Отправлен row before Создан row:\n%s", body)
	}

	rows := strings.Split(body, `<div class="receipts-row">`)
	// rows[0] — header + инструменты; далее data-строки.
	if len(rows) < 3 {
		t.Fatalf("expected 2 data rows, got %d", len(rows)-1)
	}
	sentRow, createdRow := rows[1], rows[2]

	for name, block := range map[string]string{"sent": sentRow, "created": createdRow} {
		info := indexesOf(block, `<div class="receipts-cell"`)
		if len(info) != 6 {
			t.Fatalf("%s: expected 6 info cells, got %d", name, len(info))
		}
		actions := indexesOf(block, `<div class="receipts-cell is-actions">`)
		if len(actions) != 1 || actions[0] < info[5] {
			t.Fatalf("%s: actions cell must follow the 6 info cells", name)
		}
	}

	// Отправлен: цвет только в 6-й ячейке, ровно одно вхождение #00BFFF.
	colorIdx := indexesOf(sentRow, `style="background-color: #00BFFF; color: #000000;"`)
	if len(colorIdx) != 1 {
		t.Fatalf("sent: expected exactly one #00BFFF cell style, got %d", len(colorIdx))
	}
	info := indexesOf(sentRow, `<div class="receipts-cell"`)
	if colorIdx[0] < info[5] {
		t.Fatal("sent: color must be on the 6th (status) cell only")
	}
	if !strings.Contains(sentRow, "Отправлен") {
		t.Fatalf("sent: expected status text Отправлен in the cell:\n%s", sentRow)
	}

	// Создан: без inline-стиля и без цвета.
	if strings.Contains(createdRow, `style="background-color:`) {
		t.Fatal("created: must not have inline background-color")
	}
	if !strings.Contains(createdRow, "Создан") {
		t.Fatalf("created: expected status text Создан in the cell:\n%s", createdRow)
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
