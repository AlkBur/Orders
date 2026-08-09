package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/sessions"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

func TestOrganizationsPage(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
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
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/new", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "new")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationCard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestOrganizationCard_Edit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	o := orgs.New()
	o.Name = "Test Org"
	o.UUID = "app-org-card-edit"
	if err := orgs.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/"+strconv.FormatInt(o.ID, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(o.ID, 10))
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
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
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
	}

	body := "name=New+Org&active=on"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/0",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "0")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	if got := w.Header().Get("Location"); got != "/organizations" {
		t.Fatalf("expected redirect to /organizations, got %q", got)
	}

	list, err := orgs.List(context.Background(), organizations.ListOptions{}, nil)
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
	o.UUID = "app-org-update"
	if err := orgs.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
	}

	body := "uuid=" + o.UUID + "&name=Updated"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(o.ID, 10),
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(o.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	if got := w.Header().Get("Location"); got != "/organizations" {
		t.Fatalf("expected redirect to /organizations, got %q", got)
	}

	got, err := orgs.GetByID(context.Background(), o.ID)
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
	}

	body := "name=Nope"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/999",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func newSession(store *sessions.Store, id string) *sessions.Session {
	uid := int64(1)
	session := &sessions.Session{
		ID:        id,
		UserID:    &uid,
		Values:    map[string]any{},
		UserAgent: "test",
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	store.Save(session)
	return session
}

func TestOrganizationSave_SetsFlash(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	sessionStore := sessions.NewStore(db)
	session := newSession(sessionStore, "sess-org-save")

	app := &App{
		organizations: organizations.NewStore(db),
		customers:     customers.NewStore(db),
		sessions:      sessionStore,
	}

	body := "name=New+Org&active=on"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/0", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "0")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(context.WithValue(r.Context(), sessionContextKey, session))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
	if session.Flash == nil || session.Flash.Type != sessions.FlashSuccess || session.Flash.Message != "Организация сохранена." {
		t.Fatalf("expected success flash, got %+v", session.Flash)
	}
}

func TestOrganizationsPage_ShowsFlash(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	sessionStore := sessions.NewStore(db)
	session := newSession(sessionStore, "sess-org-list")
	session.SetFlash(sessions.FlashSuccess, "Организация сохранена.")
	sessionStore.Save(session)

	app := &App{
		organizations: organizations.NewStore(db),
		customers:     customers.NewStore(db),
		sessions:      sessionStore,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	r = r.WithContext(context.WithValue(r.Context(), sessionContextKey, session))

	app.OrganizationsPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Организация сохранена.") {
		t.Fatalf("expected flash message on the page")
	}
	if session.Flash != nil {
		t.Fatalf("expected flash to be cleared in memory")
	}
	loaded, err := sessionStore.FindByID(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Flash != nil {
		t.Fatalf("expected flash to be cleared in the store")
	}
}

func TestOrganizationSave_Create_WithUUID(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
	}

	body := "uuid=explicit-uuid&name=New+Org&active=on"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/0", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "0")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	got, err := orgs.GetByUUID(context.Background(), "explicit-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "New Org" {
		t.Fatalf("expected 'New Org', got '%s'", got.Name)
	}
}

func TestOrganizationSave_ChangeUUID(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	o := orgs.New()
	o.Name = "Original"
	o.UUID = "old-uuid"
	if err := orgs.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		orgKeys: map[string]string{
			"old-uuid": o.APIKey,
		},
	}

	body := "uuid=new-uuid&name=Original"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(o.ID, 10), strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(o.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	if _, err := orgs.GetByUUID(context.Background(), "old-uuid"); err != organizations.ErrNotFound {
		t.Fatalf("expected ErrNotFound for old uuid, got %v", err)
	}
	got, err := orgs.GetByUUID(context.Background(), "new-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "new-uuid" {
		t.Fatalf("expected new-uuid, got %s", got.UUID)
	}
	if got.APIKey != o.APIKey {
		t.Fatalf("expected API key to remain unchanged, got %q", got.APIKey)
	}

	if _, ok := app.orgKeys["old-uuid"]; ok {
		t.Fatal("expected old uuid to be removed from orgKeys")
	}
	if key, ok := app.orgKeys["new-uuid"]; !ok || key != o.APIKey {
		t.Fatalf("expected orgKeys[new-uuid] == old API key, got %q, ok=%v", key, ok)
	}
}

func TestOrganizationSave_DuplicateUUID_Create(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	o := orgs.New()
	o.Name = "Existing"
	o.UUID = "dup-uuid"
	if err := orgs.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
	}

	body := "uuid=dup-uuid&name=New+Org"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/0", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "0")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "таким UUID уже существует") {
		t.Fatal("expected duplicate UUID alert on the page")
	}

	list, err := orgs.List(context.Background(), organizations.ListOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(list))
	}
}

func TestOrganizationSave_DuplicateUUID_Update(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	o1 := orgs.New()
	o1.Name = "First"
	o1.UUID = "uuid-1"
	if err := orgs.Save(context.Background(), o1); err != nil {
		t.Fatal(err)
	}

	o2 := orgs.New()
	o2.Name = "Second"
	o2.UUID = "uuid-2"
	if err := orgs.Save(context.Background(), o2); err != nil {
		t.Fatal(err)
	}

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		orgKeys: map[string]string{
			"uuid-1": o1.APIKey,
			"uuid-2": o2.APIKey,
		},
	}

	// Пытаемся сменить uuid второй организации на занятый.
	body := "uuid=uuid-1&name=Second"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(o2.ID, 10), strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(o2.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.OrganizationSave(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "таким UUID уже существует") {
		t.Fatal("expected duplicate UUID alert on the page")
	}

	// Старые данные не изменились.
	got, err := orgs.GetByID(context.Background(), o2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "uuid-2" {
		t.Fatalf("expected uuid to remain uuid-2, got %s", got.UUID)
	}
	if got.Name != "Second" {
		t.Fatalf("expected name to remain Second, got %s", got.Name)
	}
	if _, ok := app.orgKeys["uuid-2"]; !ok {
		t.Fatal("expected orgKeys to keep uuid-2")
	}
}

func TestOrganizationChangeUUID_APIAuth(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgs := organizations.NewStore(db)

	o := orgs.New()
	o.Name = "Original"
	o.UUID = "old-uuid"
	if err := orgs.Save(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	app := &App{
		organizations: orgs,
		customers:     customers.NewStore(db),
		orgKeys: map[string]string{
			"old-uuid": o.APIKey,
		},
	}

	// Меняем UUID.
	body := "uuid=new-uuid&name=Original"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/organizations/"+strconv.FormatInt(o.ID, 10), strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(o.ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	app.OrganizationSave(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	auth := func(oid, key string) int {
		handler := app.RequireOrganizationAPIKey(http.HandlerFunc(okHandler))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+oid+"/customers", nil)
		if key != "" {
			req.Header.Set("X-API-Key", key)
		}
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("oid", oid)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	// Старый UUID больше не аутентифицирует.
	if code := auth("old-uuid", o.APIKey); code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for old uuid, got %d", code)
	}
	// Новый UUID с прежним ключом работает.
	if code := auth("new-uuid", o.APIKey); code != http.StatusOK {
		t.Fatalf("expected 200 for new uuid, got %d", code)
	}
}
