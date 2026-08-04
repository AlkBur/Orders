package app

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"Orders/internal/users"
)

// msgLoginRateLimited — сообщение при превышении лимита попыток входа.
const msgLoginRateLimited = "Too many attempts. Try again later."

// loginRateLimiter ограничивает попытки входа по IP и по аккаунту.
// Лимитеры независимы: атакующий не исчерпывает IP-лимит чужим аккаунтом
// и не исчерпывает лимит одного аккаунта с разных адресов.
func (a *App) loginRateLimiter() func(http.Handler) http.Handler {
	onRateLimited := func(w http.ResponseWriter, r *http.Request) {
		a.RenderInfrastructureError(w, r, http.StatusTooManyRequests, "Too many requests", []string{msgLoginRateLimited})
	}

	ip := httprate.LimitBy(
		a.config.RateLimit.LoginByIP.Requests,
		time.Duration(a.config.RateLimit.LoginByIP.WindowSec)*time.Second,
		clientIPKey,
		httprate.WithLimitHandler(onRateLimited),
	)
	account := httprate.LimitBy(
		a.config.RateLimit.LoginByAccount.Requests,
		time.Duration(a.config.RateLimit.LoginByAccount.WindowSec)*time.Second,
		a.loginAccountKey,
		httprate.WithLimitHandler(onRateLimited),
	)

	return func(next http.Handler) http.Handler {
		return ip(account(next))
	}
}

// clientIPKey строит ключ клиента: IP из контекста chi (после
// ClientIPFromXFF) с запасным вариантом на RemoteAddr. GetClientIP возвращает
// пустую строку при прямом соединении (без заголовка XFF), поэтому нужен
// fallback. CanonicalizeIP сводит IPv6 к /64.
func clientIPKey(r *http.Request) (string, error) {
	ip := middleware.GetClientIP(r.Context())
	if ip == "" {
		ip = r.RemoteAddr
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	return httprate.CanonicalizeIP(ip), nil
}

// loginAccountKey строит ключ аккаунта: нормализованный логин. Использует ту
// же нормализацию, что и IdentityService.GetByLogin, поэтому один и тот же
// аккаунт попадает в один bucket независимо от регистра и пробелов.
// Пустой логин возвращает IP-ключ: IP-лимитер уже применён, а пустая строка
// была бы валидным ключом httprate и создала бы общий bucket для всех
// запросов без логина.
func (a *App) loginAccountKey(r *http.Request) (string, error) {
	login := users.NormalizeLogin(r.FormValue("login"))
	if login == "" {
		return clientIPKey(r)
	}
	return login, nil
}
