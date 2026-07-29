package app

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func insertOrg(t *testing.T, dbt *sql.DB, name, apiKey string) (int64, string) {
	t.Helper()
	uuid := "uuid-" + name
	res, err := dbt.Exec(`
		INSERT INTO organizations (uuid, name, api_key, active, created_at, updated_at)
		VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid, name, apiKey)
	if err != nil {
		t.Fatalf("insert org %s: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id, uuid
}

func TestCustomersPage_Global(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
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
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	c := app.customers.New()
	c.OrganizationID = orgID
	c.Name = "Test"
	c.UUID = "app-cust-page-org"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/customers", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomersPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_NewFromGlobal(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/0/customers/new", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "0")
	rctx.URLParams.Add("id", "new")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_NewInOrg(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/customers/new", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", "new")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_Edit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	c := app.customers.New()
	c.OrganizationID = orgID
	c.Name = "Edit Me"
	c.UUID = "app-cust-card-edit"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/customers/"+strconv.FormatInt(c.ID, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", strconv.FormatInt(c.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCustomerCard_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(orgID, 10)+"/customers/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", "999")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerCard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestCustomerSave_Create(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	body := `name=New+Customer`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(orgID, 10)+"/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	list, _ := app.customers.List(context.Background(), orgID)
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
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	c := app.customers.New()
	c.OrganizationID = orgID
	c.Name = "Original"
	c.UUID = "app-cust-update"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(orgID, 10)+"/customers/"+strconv.FormatInt(c.ID, 10),
		strings.NewReader("uuid="+c.UUID+"&name=Updated"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", strconv.FormatInt(c.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	got, err := app.customers.GetByExternal(context.Background(), orgID, c.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" {
		t.Fatalf("expected 'Updated', got '%s'", got.Name)
	}
}

func TestCustomerDelete(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	orgID, _ := insertOrg(t, db, "Org One", "key_org1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
	}

	c := app.customers.New()
	c.OrganizationID = orgID
	c.Name = "To Delete"
	c.UUID = "app-cust-delete"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/organizations/"+strconv.FormatInt(orgID, 10)+"/customers/"+strconv.FormatInt(c.ID, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", strconv.FormatInt(orgID, 10))
	rctx.URLParams.Add("id", strconv.FormatInt(c.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.CustomerDelete(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	if _, err := app.customers.GetByID(context.Background(), c.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}
