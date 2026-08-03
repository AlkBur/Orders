package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func TestProductsPage_Global(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	p := app.products.New()
	p.OrganizationID = orgID
	p.Name = "Global Product"
	p.Unit = "шт"
	p.UUID = "app-prod-global"
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
}

func TestProductsPage_Org(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	p := app.products.New()
	p.OrganizationID = orgID
	p.Name = "Test"
	p.Unit = "шт"
	p.UUID = "app-prod-org"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
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
	insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/0/products/new", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "0")
	rctx.URLParams.Add("id", "new")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `class="card-header"`) || !strings.Contains(body, `class="card-content"`) {
		t.Fatal("expected product card to use the shared card component")
	}
	if strings.Contains(w.Body.String(), "page-card") {
		t.Fatal("unexpected legacy product card markup")
	}
}

func TestProductCard_NewInOrg(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products/new", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", "new")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestProductCard_Edit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	p := app.products.New()
	p.OrganizationID = orgID
	p.Name = "Edit Me"
	p.Unit = "шт"
	p.UUID = "app-prod-edit"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products/"+strconv.FormatInt(p.ID, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", strconv.FormatInt(p.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestProductCard_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", "999")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductCard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestProductSave_Create(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	body := `name=New+Product&unit=шт`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}

	list, _ := app.products.List(context.Background(), orgID, products.ListOptions{}, nil)
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
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	p := app.products.New()
	p.OrganizationID = orgID
	p.Name = "Original"
	p.Unit = "шт"
	p.UUID = "app-prod-update"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products/"+strconv.FormatInt(p.ID, 10),
		strings.NewReader("uuid="+p.UUID+"&name=Updated&unit=кг"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", strconv.FormatInt(p.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	got, err := app.products.GetByExternal(context.Background(), orgID, p.UUID)
	if err != nil {
		t.Fatal(err)
	}
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
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	p := app.products.New()
	p.OrganizationID = orgID
	p.Name = "To Delete"
	p.Unit = "шт"
	p.UUID = "app-prod-delete"
	if err := app.products.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products/"+strconv.FormatInt(p.ID, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", strconv.FormatInt(p.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductDelete(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	if _, err := app.products.GetByID(context.Background(), p.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestProductSave_UpdateNotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(orgID, 10)+"/products/999",
		strings.NewReader("name=Ghost"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", "999")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ProductSave(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
