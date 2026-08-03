package app

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
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

	loc := w.Header().Get("Location")
	if loc == "" {
		t.Fatal("step 1: missing Location header")
	}
	idStr := strings.TrimPrefix(loc, "/receipts/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		t.Fatalf("step 1: invalid receipt ID from Location: %s", loc)
	}

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
