package app

import "Orders/internal/users"

// LandingURL возвращает маршрут, на который пользователь попадает сразу
// после входа в систему.
//
// Функция намеренно «тупая»: она лишь выбирает первый экран по состоянию
// аккаунта и роли и не содержит ни проверки прав, ни навигационной логики.
//
// Приоритет:
//  1. смена пароля (всегда, независимо от роли);
//  2. админ — рабочий стол;
//  3. пользователь — документы.
func LandingURL(u users.Identity) string {
	if u.NeedsPasswordSetup() {
		return RouteSetPassword
	}
	if u.IsAdmin {
		return RouteDashboard
	}
	return RouteReceipts
}
