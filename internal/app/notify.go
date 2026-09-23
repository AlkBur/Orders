package app

import (
	"context"
	"text/template"
	"time"

	"Orders/internal/gotify"
	"Orders/internal/receipts"
)

const (
	// gotifyHTTPTimeout ограничивает один HTTP-запрос к Gotify.
	gotifyHTTPTimeout = 10 * time.Second
	// gotifyDispatchTimeout ограничивает всю фоновую рассылку.
	gotifyDispatchTimeout = 10 * time.Second

	// notifyIntegrationActor — инициатор событий, пришедших из 1С через
	// Integration API: сессии пользователя в этих запросах нет.
	notifyIntegrationActor = "1С"
)

// gotifySender — минимальный контракт отправителя уведомлений. Позволяет
// подменять Gotify в тестах.
type gotifySender interface {
	Send(ctx context.Context, msg gotify.Message) error
}

// notify отправляет уведомление в фоне: пользовательская операция не должна
// ждать Gotify и не должна падать из-за него. Контекст запроса не используется —
// после ответа он отменяется, а отправка должна иметь возможность завершиться.
func (a *App) notify(msg gotify.Message) {
	if a.notifier == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), gotifyDispatchTimeout)
		defer cancel()
		if err := a.notifier.Send(ctx, msg); err != nil {
			a.log.Error().Err(err).Msg("gotify notification failed")
		}
	}()
}

// notifyDataFromReceipt заполняет общие поля сообщения. actor — инициатор
// события, а не автор документа: rec.UserLogin здесь не используется.
func notifyDataFromReceipt(rec *receipts.Receipt, actor string) notifyData {
	total := ""
	if v, err := rec.DisplayValue("Total"); err == nil {
		total = v
	}
	return notifyData{
		Number:       rec.Number,
		Date:         rec.Date.Format("02.01.2006"),
		Organization: rec.OrganizationName,
		Customer:     rec.CustomerName,
		Total:        total,
		User:         actor,
	}
}

// notifyReceiptStatus отправляет уведомление об изменении статуса документа.
// status — новое отображаемое значение; actor — инициатор события (логин
// сессии или notifyIntegrationActor).
func (a *App) notifyReceiptStatus(rec *receipts.Receipt, status, actor string) {
	data := notifyDataFromReceipt(rec, actor)
	data.Status = status
	a.sendNotify(statusTemplates, data)
}

// notifyReceiptAction отправляет уведомление об изменении действия документа.
// Отмена действия передаётся как action = "отменено".
func (a *App) notifyReceiptAction(rec *receipts.Receipt, action, actor string) {
	data := notifyDataFromReceipt(rec, actor)
	data.Action = action
	a.sendNotify(actionTemplates, data)
}

// sendNotify рендерит заголовок и тело по шаблону и ставит уведомление в фон.
// Ошибка рендера только логируется: уведомления best-effort и не влияют на
// операцию пользователя.
func (a *App) sendNotify(tmpl *template.Template, data notifyData) {
	title, err := renderNotify(tmpl, "title", data)
	if err != nil {
		a.log.Error().Err(err).Msg("gotify template render failed")
		return
	}
	body, err := renderNotify(tmpl, "body", data)
	if err != nil {
		a.log.Error().Err(err).Msg("gotify template render failed")
		return
	}
	a.notify(gotify.Message{Title: title, Message: body})
}
