package app

import (
	"net/http"

	"Orders/internal/sessions"
	"Orders/internal/ui"
)

// SetFlash записывает Flash-сообщение в сессию и персистит её.
// При отсутствии сессии (например, в тестах) — no-op.
func (a *App) SetFlash(r *http.Request, t sessions.FlashType, message string) error {
	session := CurrentSession(r)
	if session == nil {
		return nil
	}
	session.SetFlash(t, message)
	return a.sessions.Save(session)
}

// consumeFlash читает Flash-сообщение из сессии, очищает его и
// персистит очищенное состояние. Возвращает nil, если сообщения нет.
func (a *App) consumeFlash(r *http.Request) (*sessions.Flash, error) {
	session := CurrentSession(r)
	if session == nil || session.Flash == nil {
		return nil, nil
	}
	flash := session.Flash
	session.ClearFlash()
	if err := a.sessions.Save(session); err != nil {
		return nil, err
	}
	return flash, nil
}

// FlashToAlert превращает Flash-сообщение в модель представления.
// Единственное место, где транспортный тип сессии сопоставляется
// с типом Alert'а UI.
func FlashToAlert(f sessions.Flash) *ui.AlertData {
	return &ui.AlertData{
		Type:     toAlertType(f.Type),
		Messages: []string{f.Message},
	}
}

func toAlertType(t sessions.FlashType) ui.AlertType {
	switch t {
	case sessions.FlashSuccess:
		return ui.AlertSuccess
	case sessions.FlashError:
		return ui.AlertError
	case sessions.FlashWarning:
		return ui.AlertWarning
	default:
		return ui.AlertInfo
	}
}
