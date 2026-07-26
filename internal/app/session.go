package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const SessionCookie = "orders_session"

type Session struct {
	UserID int64
}

func CreateSession(w http.ResponseWriter, secret string, session Session) {

	value := fmt.Sprintf("%d", session.UserID)
	signature := sign(secret, value)

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    base64.RawURLEncoding.EncodeToString([]byte(value + ":" + signature)),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // включить после HTTPS
	})
}

func ReadSession(r *http.Request, secret string) (*Session, error) {

	cookie, err := r.Cookie(SessionCookie)
	if err != nil {
		return nil, err
	}

	data, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(string(data), ":")

	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session")
	}

	value := parts[0]
	signature := parts[1]

	if sign(secret, value) != signature {
		return nil, fmt.Errorf("invalid signature")
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}

	return &Session{
		UserID: id,
	}, nil
}

func DeleteSession(w http.ResponseWriter) {

	http.SetCookie(w, &http.Cookie{
		Name:   SessionCookie,
		Path:   "/",
		MaxAge: -1,
	})
}

func sign(secret, value string) string {

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(value))

	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
