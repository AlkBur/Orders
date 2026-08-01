package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/testutil"
	"Orders/internal/users"
)

func newDashboardApp(t *testing.T) *App {
	t.Helper()
	db := testutil.NewTestDB(t, NewSchema())
	return &App{
		organizations: organizations.NewStore(db),
		customers:     customers.NewStore(db),
		products:      products.NewStore(db),
		receipts:      receipts.NewStore(db),
		users:         users.NewStore(db),
	}
}

func TestDashboardPage(t *testing.T) {
	app := newDashboardApp(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	app.DashboardPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	for _, want := range []string{
		`href="/receipts"`,
		`href="/organizations"`,
		`href="/customers"`,
		`href="/products"`,
		`href="/users"`,
		"Работа с товарными чеками",
		"module-card-count",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("dashboard page missing %q", want)
		}
	}
}

func TestDashboardPage_Counts(t *testing.T) {
	app := newDashboardApp(t)

	org := app.organizations.New()
	org.Name = "Org One"
	org.UUID = "dash-org"
	if err := app.organizations.Save(context.Background(), org); err != nil {
		t.Fatalf("save org: %v", err)
	}

	w := httptest.NewRecorder()
	app.DashboardPage(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if want := `module-card-count">1<`; !strings.Contains(body, want) {
		t.Fatalf("dashboard page missing formatted count %q", want)
	}
}
