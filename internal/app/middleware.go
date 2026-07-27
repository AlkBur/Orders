package app

import (
	"context"
	"net/http"
	"time"

	"Orders/internal/sessions"
	"Orders/internal/users"
)

type contextKey int

const (
	userContextKey contextKey = iota
	sessionContextKey
	integrationContextKey
)

func CurrentUser(r *http.Request) *users.User {
	user, _ := r.Context().Value(userContextKey).(*users.User)
	return user
}

func CurrentSession(r *http.Request) *sessions.Session {
	session, _ := r.Context().Value(sessionContextKey).(*sessions.Session)
	return session
}

func CurrentIntegration(r *http.Request) *Integration {
	integration, _ := r.Context().Value(integrationContextKey).(*Integration)
	return integration
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

func RequireAuth(store *sessions.Store, usersStore *users.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := CurrentSession(r)
		if session == nil || session.UserID == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := usersStore.FindByID(*session.UserID)
		if err != nil {
			store.Delete(session.ID)
			DeleteSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := CurrentUser(r)
		if user == nil || !user.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequirePassword(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := CurrentUser(r)
		if user != nil && user.NeedsPasswordSetup() {
			http.Redirect(w, r, "/set-password", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireIntegration проверяет заголовок X-API-Key для доступа
// к интеграционному API (/api/integration/*).
// При успехе помещает Integration в context.
func (a *App) RequireIntegration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		integration, ok := a.integrations[key]
		if !ok {
			a.Unauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), integrationContextKey, integration)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
