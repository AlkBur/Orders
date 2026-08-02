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
	_, orgUUID := insertOrg(t, db, "Org", "k1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{orgUUID: "k1"},
	}

	body := `[{"id":"ext-1","name":"From 1C"}]`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+orgUUID+"/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "k1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", orgUUID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result customers.SyncResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	// Get org by UUID to find its int64 ID
	org, err := app.organizations.GetByUUID(context.Background(), orgUUID)
	if err != nil {
		t.Fatal(err)
	}

	got, err := app.customers.GetByExternal(context.Background(), org.ID, "ext-1")
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
	_, orgUUID := insertOrg(t, db, "Org", "k1")
	org, _ := orgs.GetByUUID(context.Background(), orgUUID)

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{orgUUID: "k1"},
	}

	// Pre-insert via sync
	body := `[{"id":"ext-1","name":"Original"}]`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+orgUUID+"/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "k1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", orgUUID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("pre-insert: expected 200, got %d", w.Code)
	}

	// Update via sync
	body = `[{"id":"ext-1","name":"Updated","active":false}]`
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+orgUUID+"/customers",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "k1")
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("oid", orgUUID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result customers.SyncResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", result.Updated)
	}

	got, _ := app.customers.GetByExternal(context.Background(), org.ID, "ext-1")
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
	_, orgUUID := insertOrg(t, db, "Org", "k1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{orgUUID: "k1"},
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
			r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+orgUUID+"/customers",
				strings.NewReader(tt.body))
			if tt.contentTyp != "" {
				r.Header.Set("Content-Type", tt.contentTyp)
			}
			r.Header.Set("X-API-Key", "k1")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("oid", orgUUID)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestCustomersAPI_SyncToDifferentOrgs(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	_, orgUUID1 := insertOrg(t, db, "Org1", "k1")
	_, orgUUID2 := insertOrg(t, db, "Org2", "k2")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{orgUUID1: "k1", orgUUID2: "k2"},
	}

	sync := func(oUUID string, body string) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+oUUID+"/customers",
			strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-API-Key", map[string]string{orgUUID1: "k1", orgUUID2: "k2"}[oUUID])
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("oid", oUUID)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("sync to %s: expected 200, got %d", oUUID, w.Code)
		}
	}

	sync(orgUUID1, `[{"id":"shared","name":"In Org1"}]`)
	sync(orgUUID2, `[{"id":"shared","name":"In Org2"}]`)

	list, err := app.customers.List(context.Background(), 0, customers.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 total, got %d", len(list))
	}
}
