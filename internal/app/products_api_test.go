package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func TestProductsAPI_SyncInsert(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org", "k1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "k1"},
	}

	body := `[{"id":"ext-1","name":"From 1C","unit":"шт"}]`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/products",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "k1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutProducts)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result products.SyncResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", result.Inserted)
	}

	got, err := app.products.Get(context.Background(), "org1", "ext-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "From 1C" || got.Unit != "шт" {
		t.Fatalf("unexpected product: %+v", got)
	}
}

func TestProductsAPI_SyncUpdate(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org", "k1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "k1"},
	}

	// Pre-insert via sync
	body := `[{"id":"ext-1","name":"Original","unit":"шт"}]`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/products",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "k1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutProducts)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("pre-insert: expected 200, got %d", w.Code)
	}

	// Update via sync
	body = `[{"id":"ext-1","name":"Updated","unit":"кг","active":false}]`
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/products",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "k1")
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("oid", "org1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutProducts)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result products.SyncResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", result.Updated)
	}

	got, _ := app.products.Get(context.Background(), "org1", "ext-1")
	if got.Name != "Updated" || got.Unit != "кг" || got.Active {
		t.Fatalf("unexpected product after update: %+v", got)
	}
}

func TestProductsAPI_ValidationErrors(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org", "k1")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "k1"},
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
			r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/org1/products",
				strings.NewReader(tt.body))
			if tt.contentTyp != "" {
				r.Header.Set("Content-Type", tt.contentTyp)
			}
			r.Header.Set("X-API-Key", "k1")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("oid", "org1")
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutProducts)).ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestProductsAPI_SyncToDifferentOrgs(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)
	insertOrg(t, db, "org1", "Org1", "k1")
	insertOrg(t, db, "org2", "Org2", "k2")

	app := &App{
		products:      products.NewStore(db),
		organizations: orgs,
		orgKeys:       map[string]string{"org1": "k1", "org2": "k2"},
	}

	sync := func(oid, body string) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+oid+"/products",
			strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-API-Key", map[string]string{"org1": "k1", "org2": "k2"}[oid])
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("oid", oid)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutProducts)).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("sync to %s: expected 200, got %d", oid, w.Code)
		}
	}

	sync("org1", `[{"id":"shared","name":"In Org1","unit":"шт"}]`)
	sync("org2", `[{"id":"shared","name":"In Org2","unit":"шт"}]`)

	list, err := app.products.List(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 total, got %d", len(list))
	}
}
