package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"Orders/internal/sessions"
	"Orders/internal/testutil"
	"Orders/internal/users"
)

func newHomeApp(t *testing.T) (*App, *sessions.Store) {
	t.Helper()
	db := testutil.NewTestDB(t, NewSchema())

	store := users.NewStore(db)
	for _, u := range []*users.User{
		{UUID: "home-admin", Login: "admin", PasswordHash: "x", IsAdmin: true},
		{UUID: "home-user", Login: "user", PasswordHash: "x"},
		{UUID: "home-needs-pass", Login: "newbie"},
	} {
		if err := store.Create(u); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	identity := users.NewIdentityService()
	if err := identity.Load(context.Background(), store); err != nil {
		t.Fatalf("load identity: %v", err)
	}

	sessionStore := sessions.NewStore(db)

	return &App{
		identity: identity,
		sessions: sessionStore,
	}, sessionStore
}

func TestHome(t *testing.T) {
	app, sessionStore := newHomeApp(t)

	t.Run("NoSession", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		app.Home(w, r)

		assertRedirect(t, w, RouteLogin)
	})

	t.Run("UnknownUser", func(t *testing.T) {
		s, err := sessionStore.Create(9999, "test")
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(context.WithValue(r.Context(), sessionContextKey, s))

		app.Home(w, r)

		assertRedirect(t, w, RouteLogin)
		if _, err := sessionStore.FindByID(s.ID); err == nil {
			t.Fatal("expected session to be deleted")
		}
	})

	users := []*users.User{
		{Login: "admin"},
		{Login: "user"},
		{Login: "newbie"},
	}

	for _, u := range users {
		t.Run("RedirectsToLanding_"+u.Login, func(t *testing.T) {
			identity, ok := app.identity.GetByLogin(u.Login)
			if !ok {
				t.Fatalf("identity for %q not found", u.Login)
			}

			s, err := sessionStore.Create(identity.ID, "test")
			if err != nil {
				t.Fatalf("create session: %v", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = r.WithContext(context.WithValue(r.Context(), sessionContextKey, s))

			app.Home(w, r)

			assertRedirect(t, w, LandingURL(identity))
		})
	}
}

func assertRedirect(t *testing.T, w *httptest.ResponseRecorder, wantLocation string) {
	t.Helper()
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other, got %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != wantLocation {
		t.Fatalf("expected Location %q, got %q", wantLocation, got)
	}
}
