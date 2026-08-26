package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"Orders/internal/testutil"
)

// newRateLimitApp создаёт App с лимитом Integration API в 1 запрос на большое
// окно (60s). Лимит задаётся явно в тестовой конфигурации, а не через
// глобальный defaultRateLimit, чтобы тест не зависел от продакшн-значений.
func newRateLimitApp(t *testing.T) (*App, string, string) {
	t.Helper()
	db := testutil.NewTestDB(t, NewSchema())
	_, org1UUID := insertOrg(t, db, "RateOrg1", "rk1")
	_, org2UUID := insertOrg(t, db, "RateOrg2", "rk2")

	app := &App{
		config: &Config{
			RateLimit: RateLimitConfig{
				IntegrationAPI: LimitConfig{Requests: 1, WindowSec: 60},
			},
		},
		orgKeys: map[string]string{
			org1UUID: "rk1",
			org2UUID: "rk2",
		},
	}

	return app, org1UUID, org2UUID
}

func TestIntegrationAPIRateLimit(t *testing.T) {
	app, orgUUID, _ := newRateLimitApp(t)

	handler := app.integrationAPIRateLimiter()(app.RequireOrganizationAPIKey(http.HandlerFunc(okHandler)))

	w1 := httptest.NewRecorder()
	r1 := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/customers", orgUUID, "", "rk1", nil)
	handler.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 on first request, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	r2 := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/customers", orgUUID, "", "rk1", nil)
	handler.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second request, got %d", w2.Code)
	}
}

func TestIntegrationAPIRateLimit_IndependentPerOrganization(t *testing.T) {
	app, org1UUID, org2UUID := newRateLimitApp(t)

	handler := app.integrationAPIRateLimiter()(app.RequireOrganizationAPIKey(http.HandlerFunc(okHandler)))

	// Первый запрос каждой организации проходит: бюджеты независимы.
	for _, tt := range []struct {
		oid, key string
	}{
		{org1UUID, "rk1"},
		{org2UUID, "rk2"},
	} {
		w := httptest.NewRecorder()
		r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+tt.oid+"/customers", tt.oid, "", tt.key, nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for first request of %s, got %d", tt.oid, w.Code)
		}
	}

	// Второй запрос org1 упирается в собственный лимит (1/60).
	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+org1UUID+"/customers", org1UUID, "", "rk1", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for second request of org1, got %d", w.Code)
	}
}
