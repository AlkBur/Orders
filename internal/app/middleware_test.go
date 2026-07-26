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

func TestRequireAPIKey(t *testing.T) {
	app := &App{
		config: &Config{
			API: APIConfig{
				Key: "test-key",
			},
		},
	}

	handler := app.RequireAPIKey(http.HandlerFunc(okHandler))

	tests := []struct {
		name       string
		apiKey     string
		wantStatus int
	}{
		{name: "NoKey", apiKey: "", wantStatus: http.StatusUnauthorized},
		{name: "WrongKey", apiKey: "wrong", wantStatus: http.StatusUnauthorized},
		{name: "ValidKey", apiKey: "test-key", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/customers", nil)
			if tt.apiKey != "" {
				r.Header.Set("X-API-Key", tt.apiKey)
			}
			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestRequireAdminAPI(t *testing.T) {
	app := &App{}

	handler := app.RequireAdminAPI(http.HandlerFunc(okHandler))

	tests := []struct {
		name       string
		user       *users.User
		wantStatus int
	}{
		{
			name:       "NoUser",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "PasswordNotSet",
			user: &users.User{
				IsAdmin:      true,
				PasswordHash: "",
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "NotAdmin",
			user: &users.User{
				IsAdmin:      false,
				PasswordHash: "hash",
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "ValidAdmin",
			user: &users.User{
				IsAdmin:      true,
				PasswordHash: "hash",
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/customers", nil)
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
