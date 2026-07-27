package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func TestCustomersAPI_SyncInsert(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	if err := orgs.Save(context.Background(), &organizations.Organization{
		UUID: "org1", Name: "Org", APIKey: "k1",
	}); err != nil {
		t.Fatal(err)
	}

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		integrations:  map[string]*Integration{"test-key": {Name: "Test"}},
	}

	body := `[{"id":"ext-1","name":"From 1C"}]`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "test-key")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.RequireIntegration(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result customers.SyncResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	got, err := app.customers.Get(context.Background(), "org1", "ext-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "From 1C" {
		t.Fatalf("expected 'From 1C', got '%s'", got.Name)
	}
}

func TestCustomersAPI_SyncUpdate(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	if err := orgs.Save(context.Background(), &organizations.Organization{
		UUID: "org1", Name: "Org", APIKey: "k1",
	}); err != nil {
		t.Fatal(err)
	}

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		integrations:  map[string]*Integration{"test-key": {Name: "Test"}},
	}

	// Pre-insert via sync
	body := `[{"id":"ext-1","name":"Original"}]`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "test-key")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.RequireIntegration(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("pre-insert: expected 200, got %d", w.Code)
	}

	// Update via sync
	body = `[{"id":"ext-1","name":"Updated","active":false}]`
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "test-key")
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.RequireIntegration(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result customers.SyncResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", result.Updated)
	}

	got, _ := app.customers.Get(context.Background(), "org1", "ext-1")
	if got.Name != "Updated" {
		t.Fatalf("expected 'Updated', got '%s'", got.Name)
	}
	if got.Active {
		t.Fatal("expected inactive")
	}
}

func TestCustomersAPI_ValidationErrors(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	if err := orgs.Save(context.Background(), &organizations.Organization{
		UUID: "org1", Name: "Org", APIKey: "k1",
	}); err != nil {
		t.Fatal(err)
	}

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		integrations:  map[string]*Integration{"test-key": {Name: "Test"}},
	}

	tests := []struct {
		name       string
		body       string
		contentTyp string
		wantStatus int
	}{
		{
			name:       "InvalidJSON",
			body:       `{bad`,
			contentTyp: "application/json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "MissingID",
			body:       `[{"name":"No ID"}]`,
			contentTyp: "application/json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "WrongContentType",
			body:       `[{"id":"a","name":"A"}]`,
			contentTyp: "text/plain",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/customers",
				strings.NewReader(tt.body))
			if tt.contentTyp != "" {
				r.Header.Set("Content-Type", tt.contentTyp)
			}
			r.Header.Set("X-API-Key", "test-key")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("oid", "org1")
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			app.RequireIntegration(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestCustomersAPI_SyncToDifferentOrgs(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	if err := orgs.Save(context.Background(), &organizations.Organization{
		UUID: "org1", Name: "Org1", APIKey: "k1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := orgs.Save(context.Background(), &organizations.Organization{
		UUID: "org2", Name: "Org2", APIKey: "k2",
	}); err != nil {
		t.Fatal(err)
	}

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		integrations:  map[string]*Integration{"test-key": {Name: "Test"}},
	}

	sync := func(oid, body string) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+oid+"/customers",
			strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-API-Key", "test-key")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("oid", oid)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		app.RequireIntegration(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("sync to %s: expected 200, got %d", oid, w.Code)
		}
	}

	sync("org1", `[{"id":"shared","name":"In Org1"}]`)
	sync("org2", `[{"id":"shared","name":"In Org2"}]`)

	list, err := app.customers.List(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 total, got %d", len(list))
	}
}
