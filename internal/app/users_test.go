package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"Orders/internal/testutil"
	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
)

func newUsersApp(t *testing.T) *App {
	t.Helper()
	db := testutil.NewTestDB(t, NewSchema())
	store := users.NewStore(db)
	identity := users.NewIdentityService()
	if err := identity.Load(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	return &App{users: store, identity: identity}
}

// addUser создаёт пользователя в store и синхронизирует IdentityService.
func addUser(t *testing.T, app *App, login string, isAdmin bool) *users.User {
	t.Helper()
	u := &users.User{
		UUID:    "u-" + login,
		Login:   login,
		Email:   login + "@example.com",
		IsAdmin: isAdmin,
	}
	if err := app.users.Create(u); err != nil {
		t.Fatal(err)
	}
	app.identity.Add(u)
	return u
}

func userDeleteRequest(t *testing.T, id string, u users.Identity) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/users/"+id+"/delete", nil)
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, u))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func renderUserCard(t *testing.T, app *App, id string) string {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/"+id, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.UserCard(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	return w.Body.String()
}

func TestUserDelete_RequiresAdmin(t *testing.T) {
	app := newUsersApp(t)
	victim := addUser(t, app, "victim", false)
	idStr := strconv.FormatInt(victim.ID, 10)

	handler := RequireAdmin(http.HandlerFunc(app.UserDelete))
	for _, user := range []users.Identity{{}, {ID: 1, Login: "operator", IsAdmin: false}} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, userDeleteRequest(t, idStr, user))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	}

	if _, ok := app.identity.GetByID(victim.ID); !ok {
		t.Fatal("non-admin request must not remove the user")
	}
}

func TestUserDelete_LastAdministrator(t *testing.T) {
	app := newUsersApp(t)
	admin := addUser(t, app, "admin", true)
	idStr := strconv.FormatInt(admin.ID, 10)

	w := httptest.NewRecorder()
	app.UserDelete(w, userDeleteRequest(t, idStr, users.Identity{ID: admin.ID, Login: "admin", IsAdmin: true}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	if _, err := app.users.GetByID(context.Background(), admin.ID); err != nil {
		t.Fatalf("last administrator must remain in store: %v", err)
	}
	if _, ok := app.identity.GetByID(admin.ID); !ok {
		t.Fatal("last administrator must remain in identity")
	}
}

func TestUserDelete_Success(t *testing.T) {
	app := newUsersApp(t)
	addUser(t, app, "admin", true)
	victim := addUser(t, app, "victim", false)
	idStr := strconv.FormatInt(victim.ID, 10)

	w := httptest.NewRecorder()
	app.UserDelete(w, userDeleteRequest(t, idStr, users.Identity{ID: 1, Login: "admin", IsAdmin: true}))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/users" {
		t.Fatalf("expected redirect to /users, got %q", loc)
	}

	if _, err := app.users.GetByID(context.Background(), victim.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected user removed from store, got %v", err)
	}
	if _, ok := app.identity.GetByID(victim.ID); ok {
		t.Fatal("expected user removed from identity")
	}
}

func TestUserDelete_AdminNotLast(t *testing.T) {
	app := newUsersApp(t)
	first := addUser(t, app, "admin1", true)
	second := addUser(t, app, "admin2", true)
	idStr := strconv.FormatInt(second.ID, 10)

	w := httptest.NewRecorder()
	app.UserDelete(w, userDeleteRequest(t, idStr, users.Identity{ID: first.ID, Login: "admin1", IsAdmin: true}))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}

	if _, ok := app.identity.GetByID(second.ID); ok {
		t.Fatal("expected deleted administrator removed from identity")
	}
	if !app.identity.IsLastAdministrator(first.ID) {
		t.Fatal("expected exactly one administrator to remain")
	}
}

func TestUserDelete_NotFound(t *testing.T) {
	app := newUsersApp(t)
	admin := addUser(t, app, "admin", true)

	w := httptest.NewRecorder()
	app.UserDelete(w, userDeleteRequest(t, "99999", users.Identity{ID: admin.ID, Login: "admin", IsAdmin: true}))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// TestUserDelete_ConcurrentKeepsAdministrator проверяет сам инвариант:
// одновременное удаление двух администраторов не может оставить систему
// без администратора.
func TestUserDelete_ConcurrentKeepsAdministrator(t *testing.T) {
	app := newUsersApp(t)
	first := addUser(t, app, "admin1", true)
	second := addUser(t, app, "admin2", true)

	requests := []*http.Request{
		userDeleteRequest(t, strconv.FormatInt(first.ID, 10), users.Identity{ID: first.ID, Login: "admin1", IsAdmin: true}),
		userDeleteRequest(t, strconv.FormatInt(second.ID, 10), users.Identity{ID: second.ID, Login: "admin2", IsAdmin: true}),
	}

	codes := make([]int, len(requests))
	var wg sync.WaitGroup
	for i, req := range requests {
		wg.Add(1)
		go func(i int, req *http.Request) {
			defer wg.Done()
			w := httptest.NewRecorder()
			app.UserDelete(w, req)
			codes[i] = w.Code
		}(i, req)
	}
	wg.Wait()

	successes := 0
	for _, code := range codes {
		if code == http.StatusSeeOther {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one successful delete, got %d (%v)", successes, codes)
	}

	remainingAdmins := 0
	for _, u := range []*users.User{first, second} {
		if _, ok := app.identity.GetByID(u.ID); ok {
			remainingAdmins++
		}
	}
	if remainingAdmins != 1 {
		t.Fatalf("expected exactly one administrator to remain, got %d", remainingAdmins)
	}
}

func TestUserSave_CannotRevokeLastAdmin(t *testing.T) {
	app := newUsersApp(t)
	admin := addUser(t, app, "admin", true)
	idStr := strconv.FormatInt(admin.ID, 10)

	form := url.Values{}
	form.Set("uuid", admin.UUID)
	form.Set("login", admin.Login)
	// is_admin не передан → попытка снять права единственного администратора.

	r := httptest.NewRequest(http.MethodPost, "/users/"+idStr, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	app.UserSave(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if identity, ok := app.identity.GetByID(admin.ID); !ok || !identity.IsAdmin {
		t.Fatal("last administrator must keep admin rights")
	}
}

func TestUserCard_DeleteButton(t *testing.T) {
	app := newUsersApp(t)
	admin := addUser(t, app, "admin", true)
	regular := addUser(t, app, "regular", false)

	adminBody := renderUserCard(t, app, strconv.FormatInt(admin.ID, 10))
	if strings.Contains(adminBody, "/delete") {
		t.Fatalf("last administrator must not have delete button:\n%s", adminBody)
	}

	regularID := strconv.FormatInt(regular.ID, 10)
	regularBody := renderUserCard(t, app, regularID)
	if !strings.Contains(regularBody, `action="/users/`+regularID+`/delete"`) {
		t.Fatalf("expected delete form for regular user:\n%s", regularBody)
	}
	if !strings.Contains(regularBody, "data-confirm=") {
		t.Fatalf("expected data-confirm on delete form:\n%s", regularBody)
	}

	newBody := renderUserCard(t, app, "new")
	if strings.Contains(newBody, "/delete") {
		t.Fatalf("new user must not have delete button:\n%s", newBody)
	}
}
