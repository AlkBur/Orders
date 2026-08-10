package app

import (
	"time"

	"Orders/internal/receipts"
)

// StatusKey — семантический идентификатор состояния статуса для UI.
// Не является цветом: фактический цвет выбирает CSS в зависимости
// от активной темы. Значение используется как CSS-класс
// (receipts-status is-<key>) и может применяться другими средствами UI.
type StatusKey string

const (
	StatusKeyCreated   StatusKey = "created"
	StatusKeyOverdue   StatusKey = "overdue"
	StatusKeySent      StatusKey = "sent"
	StatusKeyAccepted  StatusKey = "accepted"
	StatusKeyCancelled StatusKey = "cancelled"
	StatusKeyProcessed StatusKey = "processed"
	StatusKeyFinished  StatusKey = "finished"
)

// StatusPresentation — готовая к выводу презентация статуса чека:
// отображаемый текст и семантический ключ для CSS.
type StatusPresentation struct {
	Display   string
	StatusKey StatusKey
}

// statusKeyByStatus сопоставляет фактический статус документа
// семантическому UI-ключу. Покрываются все фиксированные статусы
// и производное состояние OverdueStatus.
var statusKeyByStatus = map[string]StatusKey{
	receipts.StatusCreated:   StatusKeyCreated,
	receipts.StatusSent:      StatusKeySent,
	receipts.StatusAccepted:  StatusKeyAccepted,
	receipts.StatusCancelled: StatusKeyCancelled,
	receipts.StatusProcessed: StatusKeyProcessed,
	receipts.StatusFinished:  StatusKeyFinished,
	receipts.OverdueStatus:   StatusKeyOverdue,
}

// statusPresentation готовит отображение статуса документа из фактического
// значения в БД и времени создания (created_at).
//
// Пустой статус — только legacy-данные из старой базы: для совместимости
// документ с sent_at трактуется как StatusSent, иначе — StatusCreated.
// Новые документы всегда хранят фиксированный статус.
//
// StatusCreated старше 24 часов (created_at + 24h <= now) отображается как
// производное состояние OverdueStatus. В базе статус не меняется;
// как только документ перестаёт быть StatusCreated, просрочка исчезает.
func statusPresentation(status string, createdAt time.Time, sentAt *time.Time) StatusPresentation {
	st := status
	if st == "" && sentAt != nil {
		st = receipts.StatusSent
	} else if st == "" {
		st = receipts.StatusCreated
	}

	if st == receipts.StatusCreated && overdue(createdAt) {
		st = receipts.OverdueStatus
	}

	key, ok := statusKeyByStatus[st]
	if !ok {
		return StatusPresentation{Display: st}
	}
	return StatusPresentation{Display: st, StatusKey: key}
}

// overdue определяет, что документ со статусом StatusCreated является
// просроченным: с момента создания прошло не менее 24 часов.
// Нулевое значение createdAt (не прочитано из БД) просрочкой не считается.
func overdue(createdAt time.Time) bool {
	if createdAt.IsZero() {
		return false
	}
	deadline := createdAt.Add(24 * time.Hour)
	return !deadline.After(time.Now())
}