package app

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/common"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func loadPageTemplates(t *testing.T, a *App, page string) {
	t.Helper()
	tmpl, err := LoadTemplates(page)
	if err != nil {
		t.Fatalf("failed to load template %q: %v", page, err)
	}
	if a.templates == nil {
		a.templates = make(map[string]*template.Template)
	}
	a.templates[page] = tmpl
}

func TestProductsPage_Global(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}
	loadPageTemplates(t, app, "products")

	p := app.products.New()
	p.OrganizationID = "org1"
	p.Name = "Global Product"
	p.Unit = "шт"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/products", nil)
	app.ProductsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Организация") {
		t.Fatal("expected org column in global product list")
	}
	if !strings.Contains(body, "Активен") {
		t.Fatal("expected active column in global product list")
	}
	if !strings.Contains(body, "/organizations/org1/products/"+p.ID) {
		t.Fatal("expected product URL with org1, got nil UUID")
	}
}

func TestProductsPage_Org(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}
	loadPageTemplates(t, app, "products")

	p := app.products.New()
	p.OrganizationID = "org1"
	p.Name = "Test"
	p.Unit = "шт"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/products", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "Организация") {
		t.Fatal("unexpected org column in org-scoped product list")
	}
	if !strings.Contains(body, "Активен") {
		t.Fatal("expected active column in org-scoped product list")
	}
}

func TestProductCard_NewFromGlobal(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+common.NilUUID+"/products/"+common.NilUUID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", common.NilUUID)
	rctx.URLParams.Add("id", common.NilUUID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestProductCard_NewInOrg(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/products/"+common.NilUUID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", common.NilUUID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestProductCard_Edit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	p := app.products.New()
	p.OrganizationID = "org1"
	p.Name = "Edit Me"
	p.Unit = "шт"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/products/"+p.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", p.ID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestProductCard_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/products/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", "nonexistent")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestProductSave_Create(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	body := `name=New+Product&unit=шт&id=` + common.NilUUID
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/org1/products",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}

	list, _ := app.products.List(context.Background(), "org1")
	if len(list) != 1 {
		t.Fatalf("expected 1 product, got %d", len(list))
	}
	if list[0].Name != "New Product" {
		t.Fatalf("expected 'New Product', got '%s'", list[0].Name)
	}
	if list[0].Unit != "шт" {
		t.Fatalf("expected 'шт', got '%s'", list[0].Unit)
	}
}

func TestProductSave_Update(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	p := app.products.New()
	p.OrganizationID = "org1"
	p.Name = "Original"
	p.Unit = "шт"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	body := "id=" + p.ID + "&name=Updated&unit=кг"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/org1/products",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	got, _ := app.products.Get(context.Background(), "org1", p.ID)
	if got.Name != "Updated" {
		t.Fatalf("expected 'Updated', got '%s'", got.Name)
	}
	if got.Unit != "кг" {
		t.Fatalf("expected 'кг', got '%s'", got.Unit)
	}
}

func TestProductDelete(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	p := app.products.New()
	p.OrganizationID = "org1"
	p.Name = "To Delete"
	p.Unit = "шт"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/organizations/org1/products/"+p.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", p.ID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductDelete(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	if _, err := app.products.Get(context.Background(), "org1", p.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestProductSave_UpdateNotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	body := "id=nonexistent&name=Ghost"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/org1/products",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductSave(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}


