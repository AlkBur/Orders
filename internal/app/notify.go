package app

import (
	"context"
	"time"

	"Orders/internal/gotify"
	"Orders/internal/receipts"
)

const (
	// gotifyHTTPTimeout ограничивает один HTTP-запрос к Gotify.
	gotifyHTTPTimeout = 10 * time.Second
	// gotifyDispatchTimeout ограничивает всю фоновую рассылку.
	gotifyDispatchTimeout = 10 * time.Second
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

// receiptNotifyBody — общее тело уведомления по документу. actor — логин
// пользователя, инициатора события (нажал «Отправить» / изменил действие),
// а не автора документа: rec.UserLogin здесь не используется.
func receiptNotifyBody(rec *receipts.Receipt, actor string) string {
	total := ""
	if v, err := rec.DisplayValue("Total"); err == nil {
		total = v
	}
	return "Организация: " + rec.OrganizationName + "\n" +
		"Клиент: " + rec.CustomerName + "\n" +
		"Сумма: " + total + "\n" +
		"Пользователь: " + actor
}

// notifyReceiptSent — уведомление об успешной отправке документа в бухгалтерию.
// actor — пользователь, нажавший «Отправить»; он может отличаться от автора.
func notifyReceiptSent(rec *receipts.Receipt, actor string) gotify.Message {
	return gotify.Message{
		Title:   "Чек №" + rec.Number + " отправлен в бухгалтерию",
		Message: receiptNotifyBody(rec, actor),
	}
}

// notifyReceiptAction — уведомление об изменении действия документа. actor —
// пользователь, изменивший действие; он может отличаться от автора. Пустое
// action означает отмену ранее установленного действия.
func notifyReceiptAction(rec *receipts.Receipt, action, actor string) gotify.Message {
	if action == "" {
		return gotify.Message{
			Title:   "Чек №" + rec.Number + ": действие отменено",
			Message: receiptNotifyBody(rec, actor) + "\nДействие: отменено",
		}
	}
	return gotify.Message{
		Title:   "Чек №" + rec.Number + ": действие " + action,
		Message: receiptNotifyBody(rec, actor) + "\nДействие: " + action,
	}
}
