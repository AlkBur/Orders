package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/sessions"
	"Orders/internal/testutil"
	"Orders/internal/users"
)

// newBasePathApp собирает полное приложение с заданным BasePath.
// Хранилища работают на временной SQLite-базе; админ подсеивается.
func newBasePathApp(t *testing.T, basePath string) *App {
	t.Helper()

	db := testutil.NewTestDB(t, NewSchema())

	userStore := users.NewStore(db)
	if err := users.Seed(userStore); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	// Настраиваем пароль администратора, чтобы LandingURL вёл на dashboard,
	// а не на обязательную смену пароля.
	admin, err := userStore.FindByLogin("admin")
	if err != nil {
		t.Fatalf("find admin: %v", err)
	}
	if err := admin.SetPassword("admin"); err != nil {
		t.Fatalf("set admin password: %v", err)
	}
	if err := userStore.Update(admin); err != nil {
		t.Fatalf("update admin: %v", err)
	}

	identity := users.NewIdentityService()
	if err := identity.Load(context.Background(), userStore); err != nil {
		t.Fatalf("load identity: %v", err)
	}

	orgStore := organizations.NewStore(db)
	orgKeys, err := orgStore.LoadAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("load api keys: %v", err)
	}

	filesDB := testutil.NewTestDB(t, NewFilesSchema())

	app := &App{
		config:        &Config{BasePath: NormalizeBasePath(basePath), Auth: AuthConfig{InitialPassword: "admin"}},
		log:           NewLogger(false),
		db:            db,
		filesDB:       filesDB,
		users:         userStore,
		identity:      identity,
		sessions:      sessions.NewStore(db),
		customers:     customers.NewStore(db),
		organizations: orgStore,
		products:      products.NewStore(db),
		receipts:      receipts.NewStore(db),
		receiptFiles:  receipts.NewFileStore(filesDB),
		orgKeys:       orgKeys,
	}

	app.router = app.NewRouter()
	return app
}

func TestBasePath_Smoke(t *testing.T) {
	app := newBasePathApp(t, "/office")
	orgID, orgUUID := insertOrg(t, app.db, "Org", "k1")
	_ = orgID
	_ = orgUUID

	status := func(t *testing.T, method, path string) int {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, nil)
		app.Handler().ServeHTTP(w, r)
		return w.Code
	}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		// Под префиксом всё приложение доступно.
		{name: "PrefixedRoot", method: http.MethodGet, path: "/office", wantStatus: http.StatusSeeOther},
		{name: "PrefixedRootSlash", method: http.MethodGet, path: "/office/", wantStatus: http.StatusSeeOther},
		{name: "PrefixedLogin", method: http.MethodGet, path: "/office/login", wantStatus: http.StatusOK},
		{name: "PrefixedStaticCSS", method: http.MethodGet, path: "/office/static/css/bulma.min.css", wantStatus: http.StatusOK},
		{name: "PrefixedStaticJS", method: http.MethodGet, path: "/office/static/js/htmx.min.js", wantStatus: http.StatusOK},
		{name: "PrefixedCatalog", method: http.MethodGet, path: "/office/ui", wantStatus: http.StatusOK},

		// Старые адреса без префикса недоступны.
		{name: "NoPrefixRoot", method: http.MethodGet, path: "/", wantStatus: http.StatusNotFound},
		{name: "NoPrefixLogin", method: http.MethodGet, path: "/login", wantStatus: http.StatusNotFound},
		{name: "NoPrefixReceipts", method: http.MethodGet, path: "/receipts", wantStatus: http.StatusNotFound},
		{name: "NoPrefixStatic", method: http.MethodGet, path: "/static/css/bulma.min.css", wantStatus: http.StatusNotFound},
		{name: "NoPrefixCatalog", method: http.MethodGet, path: "/ui", wantStatus: http.StatusNotFound},

		// Двойной префикс и похожий префикс не должны срабатывать.
		{name: "DoublePrefix", method: http.MethodGet, path: "/office/office/login", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status(t, tt.method, tt.path); got != tt.wantStatus {
				t.Fatalf("%s %s = %d, want %d", tt.method, tt.path, got, tt.wantStatus)
			}
		})
	}
}

func TestBasePath_AuthFlow(t *testing.T) {
	app := newBasePathApp(t, "/office")

	t.Run("LoginRedirectsToPrefixedLanding", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/office/login", strings.NewReader("login=admin&password=admin"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		app.Handler().ServeHTTP(w, r)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("login status = %d, want %d", w.Code, http.StatusSeeOther)
		}
		loc := w.Header().Get("Location")
		if !strings.HasPrefix(loc, "/office/") {
			t.Fatalf("login redirect %q does not start with /office/", loc)
		}
		cookies := w.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("no session cookie set")
		}

		// Подписанная сессия открывает /office/dashboard и /office/receipts.
		for _, path := range []string{"/office/dashboard", "/office/receipts"} {
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, path, nil)
			r2.AddCookie(cookies[0])
			app.Handler().ServeHTTP(w2, r2)
			if w2.Code != http.StatusOK {
				t.Fatalf("GET %s with session = %d, want 200", path, w2.Code)
			}
		}

		// Старый адрес с валидной сессией всё равно 404.
		w3 := httptest.NewRecorder()
		r3 := httptest.NewRequest(http.MethodGet, "/receipts", nil)
		r3.AddCookie(cookies[0])
		app.Handler().ServeHTTP(w3, r3)
		if w3.Code != http.StatusNotFound {
			t.Fatalf("GET /receipts with session = %d, want 404", w3.Code)
		}
	})
}

func TestBasePath_ApiBoundary(t *testing.T) {
	app := newBasePathApp(t, "/office")
	_, orgUUID := insertOrg(t, app.db, "Org", "k1")
	app.orgKeys[orgUUID] = "k1"

	do := func(t *testing.T, path string, keys map[string]string) int {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, path, strings.NewReader(`[]`))
		for k, v := range keys {
			r.Header.Set(k, v)
		}
		app.Handler().ServeHTTP(w, r)
		return w.Code
	}

	tests := []struct {
		name       string
		path       string
		keys       map[string]string
		wantStatus int
	}{
		{
			name:       "PrefixedWithKey",
			path:       "/office/api/integration/organizations/" + orgUUID + "/customers",
			keys:       map[string]string{"Content-Type": "application/json", "X-API-Key": "k1"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "PrefixedWithoutKey",
			path:       "/office/api/integration/organizations/" + orgUUID + "/customers",
			keys:       map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "NoPrefix",
			path:       "/api/integration/organizations/" + orgUUID + "/customers",
			keys:       map[string]string{"Content-Type": "application/json", "X-API-Key": "k1"},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := do(t, tt.path, tt.keys); got != tt.wantStatus {
				t.Fatalf("%s = %d, want %d", tt.path, got, tt.wantStatus)
			}
		})
	}
}

func TestBasePath_OfficeXYZ(t *testing.T) {
	app := newBasePathApp(t, "/office")

	tests := []string{
		"/officeXYZ/receipts",
		"/officeXYZ/login",
		"/officeXYZ/",
		"/officeXYZ",
	}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, path, nil)
			app.Handler().ServeHTTP(w, r)
			if w.Code != http.StatusNotFound {
				t.Fatalf("GET %s = %d, want 404", path, w.Code)
			}
		})
	}
}

func TestLogoutRedirectIncludesBasePath(t *testing.T) {
	app := newBasePathApp(t, "/office")

	// Создаём сессию через логин, чтобы logout прошёл мимо RequireAuth.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/office/login", strings.NewReader("login=admin&password=admin"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("login status = %d, want 303", w.Code)
	}
	session := w.Result().Cookies()[0]

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPost, "/office/logout", nil)
	r2.AddCookie(session)
	app.Handler().ServeHTTP(w2, r2)

	if w2.Code != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want 303", w2.Code)
	}
	if loc := w2.Header().Get("Location"); loc != "/office/" {
		t.Fatalf("logout redirect = %q, want %q", loc, "/office/")
	}
}

func TestURL_BuildsFullPath(t *testing.T) {
	// Двух экземпляров в одном процессе не бывает, но контракт а.URL
	// обязан быть независимым: префикс — свойство инстанса, не маршрута.
	named := []struct {
		basePath string
		in       string
		want     string
	}{
		{"", "/receipts/1", "/receipts/1"},
		{"/", "/receipts/1", "/receipts/1"},
		{"/office", "/receipts/1", "/office/receipts/1"},
		{"/office/", "/receipts/1", "/office/receipts/1"},
	}

	for _, tt := range named {
		t.Run(tt.basePath+"_"+tt.in, func(t *testing.T) {
			app := newBasePathApp(t, tt.basePath)
			if got := app.URL(tt.in); got != tt.want {
				t.Fatalf("a.URL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	// Неабсолютный путь контрактом не является — возвращается как есть.
	if got := (&App{config: &Config{BasePath: "/office"}}).URL("receipts/1"); got != "receipts/1" {
		t.Fatalf("non-absolute path mangled: %q", got)
	}
}

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/", ""},
		{"app", "/app"},
		{"/app", "/app"},
		{"/app/", "/app"},
		{" /app/ ", "/app"},
	}
	for _, tt := range tests {
		if got := NormalizeBasePath(tt.in); got != tt.want {
			t.Fatalf("NormalizeBasePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBasePathStaticContent(t *testing.T) {
	app := newBasePathApp(t, "/office")

	tests := []struct {
		name       string
		path       string
		contentKey string
	}{
		{name: "CSS", path: "/office/static/css/bulma.min.css", contentKey: "Pico CSS"},
		{name: "JS", path: "/office/static/js/htmx.min.js", contentKey: "htmx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)
			app.Handler().ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("%s = %d, want 200", tt.path, w.Code)
			}
			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatal(err)
			}
			if len(body) == 0 {
				t.Fatalf("%s: empty body", tt.path)
			}
		})
	}
}

func TestBasePathHTMLContainsNoBareURLs(t *testing.T) {
	app := newBasePathApp(t, "/office")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/office/login", strings.NewReader("login=admin&password=admin"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("login status = %d", w.Code)
	}
	session := w.Result().Cookies()[0]

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/office/receipts", nil)
	r2.AddCookie(session)
	app.Handler().ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("receipts status = %d, want 200", w2.Code)
	}

	body := w2.Body.String()
	for _, bare := range []string{`href="/static`, `src="/static`, `href="/receipts`, `action="/login`} {
		if strings.Contains(body, bare) {
			t.Fatalf("page contains bare URL %q without prefix", bare)
		}
	}
}
