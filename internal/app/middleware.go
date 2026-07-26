package app

import (
	"context"
	"net/http"
	"time"

	"Orders/internal/sessions"
	"Orders/internal/users"
)

type contextKey string

const userContextKey contextKey = "user"
const sessionContextKey contextKey = "session"

func CurrentUser(r *http.Request) *users.User {
	user, _ := r.Context().Value(userContextKey).(*users.User)
	return user
}

func CurrentSession(r *http.Request) *sessions.Session {
	session, _ := r.Context().Value(sessionContextKey).(*sessions.Session)
	return session
}

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
