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

func TestOrganizationsPage(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		integrations:  make(map[string]*Integration),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	app.OrganizationsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestOrganizationCard_New(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		integrations:  make(map[string]*Integration),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/00000000-0000-0000-0000-000000000000", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "00000000-0000-0000-0000-000000000000")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestOrganizationCard_Edit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org-uuid", "Test Org", "test-key")

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		integrations:  make(map[string]*Integration),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/org-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "org-uuid")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestOrganizationCard_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		integrations:  make(map[string]*Integration),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationCard(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestOrganizationSave_Create(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		integrations:  make(map[string]*Integration),
	}

	body := "name=New+Org&active=on"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/00000000-0000-0000-0000-000000000000",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "00000000-0000-0000-0000-000000000000")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	list, err := orgs.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(list))
	}
	if list[0].Name != "New Org" {
		t.Fatalf("expected 'New Org', got '%s'", list[0].Name)
	}
}

func TestOrganizationSave_Update(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	o := orgs.New()
	o.Name = "Original"
	if err := orgs.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		integrations:  make(map[string]*Integration),
	}

	body := "name=Updated"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+o.UUID,
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", o.UUID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	got, err := orgs.Get(context.Background(), o.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" {
		t.Fatalf("expected 'Updated', got '%s'", got.Name)
	}
}

func TestOrganizationSave_UpdateNotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		integrations:  make(map[string]*Integration),
	}

	body := "name=Nope"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/nonexistent",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
