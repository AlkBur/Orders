package app

import (
	"context"
	"net/http"
	"time"

	"Orders/internal/sessions"
	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
)

type contextKey int

const (
	userContextKey contextKey = iota
	sessionContextKey
)

func CurrentUser(r *http.Request) users.Identity {
	user, _ := r.Context().Value(userContextKey).(users.Identity)
	return user
}

func CurrentSession(r *http.Request) *sessions.Session {
	session, _ := r.Context().Value(sessionContextKey).(*sessions.Session)
	return session
}

// SessionMiddleware извлекает сессию из cookie и помещает её в context.
// Он никогда не принимает решений об авторизации — только загрузка контекста.
func SessionMiddleware(store *sessions.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookie)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			session, err := store.FindByID(cookie.Value)
			if err != nil || time.Now().After(session.ExpiresAt) {
				if session != nil {
					store.Delete(session.ID)
				}
				DeleteSessionCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			store.Touch(session)

			ctx := context.WithValue(r.Context(), sessionContextKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAuth(store *sessions.Store, identity *users.IdentityService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := CurrentSession(r)
		if session == nil || session.UserID == nil {
			http.Redirect(w, r, RouteLogin, http.StatusSeeOther)
			return
		}

		user, ok := identity.GetByID(*session.UserID)
		if !ok {
			store.Delete(session.ID)
			DeleteSessionCookie(w)
			http.Redirect(w, r, RouteLogin, http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := CurrentUser(r)
		if user.ID == 0 || !user.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequirePassword(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := CurrentUser(r)
		if user.ID != 0 && user.NeedsPasswordSetup() {
			http.Redirect(w, r, RouteSetPassword, http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireOrganizationAPIKey проверяет X-API-Key для доступа
// к интеграционному API (/api/integration/organizations/{oid}/*).
// Ключ сверяется с api_key организации, извлечённой из URL через {oid}.
func (a *App) RequireOrganizationAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oid := chi.URLParam(r, "oid")
		if oid == "" {
			a.Unauthorized(w)
			return
		}

		a.orgKeysMu.RLock()
		expected, ok := a.orgKeys[oid]
		a.orgKeysMu.RUnlock()

		if !ok {
			a.Unauthorized(w)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key != expected {
			a.Unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}
