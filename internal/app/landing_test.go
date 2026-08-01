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

func TestLandingURL(t *testing.T) {
	tests := []struct {
		name string
		user users.Identity
		want string
	}{
		{
			name: "NeedsPasswordSetupWinsOverAdmin",
			user: users.Identity{ID: 1, Login: "admin", IsAdmin: true},
			want: RouteSetPassword,
		},
		{
			name: "Admin",
			user: users.Identity{ID: 2, Login: "admin", PasswordHash: "x", IsAdmin: true},
			want: RouteDashboard,
		},
		{
			name: "RegularUser",
			user: users.Identity{ID: 3, Login: "user", PasswordHash: "x"},
			want: RouteReceipts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LandingURL(tt.user); got != tt.want {
				t.Fatalf("LandingURL(%+v) = %q, want %q", tt.user, got, tt.want)
			}
		})
	}
}

func TestLoginPage_RedirectsToLanding(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())

	store := users.NewStore(db)
	user := &users.User{
		UUID:         "landing-admin",
		Login:        "admin",
		PasswordHash: "x",
		IsAdmin:      true,
	}
	if err := store.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	identity := users.NewIdentityService()
	if err := identity.Load(context.Background(), store); err != nil {
		t.Fatalf("load identity: %v", err)
	}

	app := &App{
		identity: identity,
	}

	session, err := sessions.NewStore(db).Create(user.ID, "test")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	ctx := context.WithValue(r.Context(), sessionContextKey, session)
	r = r.WithContext(ctx)

	app.LoginPage(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != RouteDashboard {
		t.Fatalf("expected redirect to %q, got %q", RouteDashboard, got)
	}
}
