package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func TestCustomersPage_Global(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/customers", nil)
	app.CustomersPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomersPage_Org(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	c := app.customers.New()
	c.OrganizationID = "org1"
	c.Name = "Test"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/customers", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomersPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_NewFromGlobal(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/00000000-0000-0000-0000-000000000000/customers/00000000-0000-0000-0000-000000000000", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "00000000-0000-0000-0000-000000000000")
	rctx.URLParams.Add("id", "00000000-0000-0000-0000-000000000000")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_NewInOrg(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/customers/00000000-0000-0000-0000-000000000000", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", "00000000-0000-0000-0000-000000000000")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_Edit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	c := app.customers.New()
	c.OrganizationID = "org1"
	c.Name = "Edit Me"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/customers/"+c.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", c.ID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org1/customers/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", "nonexistent")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestCustomerSave_Create(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	body := `name=New+Customer&id=00000000-0000-0000-0000-000000000000`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/org1/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	list, _ := app.customers.List(context.Background(), "org1")
	if len(list) != 1 {
		t.Fatalf("expected 1 customer, got %d", len(list))
	}
	if list[0].Name != "New Customer" {
		t.Fatalf("expected 'New Customer', got '%s'", list[0].Name)
	}
}

func TestCustomerSave_Update(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	c := app.customers.New()
	c.OrganizationID = "org1"
	c.Name = "Original"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	body := "id=" + c.ID + "&name=Updated"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/org1/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	got, _ := app.customers.Get(context.Background(), "org1", c.ID)
	if got.Name != "Updated" {
		t.Fatalf("expected 'Updated', got '%s'", got.Name)
	}
}

func TestCustomerDelete(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "key_org1"},
	}

	c := app.customers.New()
	c.OrganizationID = "org1"
	c.Name = "To Delete"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/organizations/org1/customers/"+c.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	rctx.URLParams.Add("id", c.ID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerDelete(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	if _, err := app.customers.Get(context.Background(), "org1", c.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}
