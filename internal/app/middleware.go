package app

import (
	"context"
	"net/http"

	"Orders/internal/users"
)

type contextKey string

const userContextKey contextKey = "user"

func RequireAuth(store *users.Store, secret string, next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		session, err := ReadSession(r, secret)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := store.FindByID(session.UserID)
		if err != nil {
			DeleteSession(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAdmin(store *users.Store, secret string, next http.Handler) http.Handler {

	return RequireAuth(store, secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		user := CurrentUser(r)

		if user == nil || !user.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

func CurrentUser(r *http.Request) *users.User {

	user, _ := r.Context().Value(userContextKey).(*users.User)
	return user
}
