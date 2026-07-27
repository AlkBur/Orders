package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"Orders/internal/users"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRequireIntegration(t *testing.T) {
	app := &App{
		integrations: map[string]*Integration{
			"test-key": {Name: "Test"},
		},
	}

	tests := []struct {
		name       string
		apiKey     string
		wantStatus int
		wantName   string
	}{
		{
			name:       "NoKey",
			apiKey:     "",
			wantStatus: http.StatusUnauthorized,
			wantName:   "",
		},
		{
			name:       "WrongKey",
			apiKey:     "wrong",
			wantStatus: http.StatusUnauthorized,
			wantName:   "",
		},
		{
			name:       "ValidKey",
			apiKey:     "test-key",
			wantStatus: http.StatusOK,
			wantName:   "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRequest *http.Request

			handler := app.RequireIntegration(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotRequest = r
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/integration/customers", nil)
			if tt.apiKey != "" {
				r.Header.Set("X-API-Key", tt.apiKey)
			}
			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantName != "" {
				integration := CurrentIntegration(gotRequest)
				if integration == nil {
					t.Fatal("expected Integration in context")
				}
				if integration.Name != tt.wantName {
					t.Fatalf("expected integration name %q, got %q", tt.wantName, integration.Name)
				}
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
