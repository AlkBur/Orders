package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRequireOrganizationAPIKey(t *testing.T) {
	app := &App{
		orgKeys: map[string]string{
			"org1": "valid-key",
		},
	}

	tests := []struct {
		name       string
		oid        string
		apiKey     string
		wantStatus int
	}{
		{
			name:       "NoKey",
			oid:        "org1",
			apiKey:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "WrongKey",
			oid:        "org1",
			apiKey:     "wrong",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "ValidKey",
			oid:        "org1",
			apiKey:     "valid-key",
			wantStatus: http.StatusOK,
		},
		{
			name:       "UnknownOrg",
			oid:        "nonexistent",
			apiKey:     "valid-key",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "EmptyOID",
			oid:        "",
			apiKey:     "valid-key",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := app.RequireOrganizationAPIKey(http.HandlerFunc(okHandler))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/integration/organizations/"+tt.oid+"/customers", nil)
			if tt.apiKey != "" {
				r.Header.Set("X-API-Key", tt.apiKey)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("oid", tt.oid)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		user       *users.User
		wantStatus int
	}{
		{
			name:       "NoUser",
			user:       nil,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "NonAdminUser",
			user:       &users.User{ID: 1, Login: "user", IsAdmin: false},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "AdminUser",
			user:       &users.User{ID: 2, Login: "admin", IsAdmin: true},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireAdmin(http.HandlerFunc(okHandler))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/customers", nil)

			if tt.user != nil {
				ctx := context.WithValue(r.Context(), userContextKey, tt.user)
				r = r.WithContext(ctx)
			}

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
