package app

import (
	"time"

	"Orders/internal/receipts"
)

// StatusPresentation — готовая к выводу презентация статуса чека:
// отображаемый текст и цвета фона/текста ячейки «Статус».
type StatusPresentation struct {
	Display string
	BG      string
	Text    string
}

// statusColors — фиксированные цвета статусов. Цвет определяется сервером
// по статусу, не зависит от темы и не может быть передан из 1С.
// Пустой фон означает отсутствие окраски.
var statusColors = map[string]struct{ bg, text string }{
	receipts.StatusCreated:   {"", ""},
	receipts.StatusSent:      {"#00BFFF", "#000000"},
	receipts.StatusAccepted:  {"#FFA500", "#000000"},
	receipts.StatusCancelled: {"#FF0000", "#FFFFFF"},
	receipts.StatusProcessed: {"#00FF7F", "#000000"},
	receipts.StatusFinished:  {"#006400", "#FFFFFF"},
	receipts.OverdueStatus:   {"#FFFF00", "#000000"},
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

	c, ok := statusColors[st]
	if !ok {
		return StatusPresentation{Display: st}
	}
	return StatusPresentation{Display: st, BG: c.bg, Text: c.text}
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