package receipts

// Статусы товарного чека. Список фиксированный: сервер сам определяет
// цвет статуса по этому списку, 1С передаёт только эти значения.
// «Просрочен» (StatusOverdue) — производное отображаемое состояние,
// в базе и API не существует: вычисляется из StatusCreated + createdAt.
const (
	StatusCreated   = "Создан"
	StatusSent      = "Отправлен"
	StatusAccepted  = "Принят"
	StatusCancelled = "Отменен"
	StatusProcessed = "Обработан"
	StatusFinished  = "Завершен"
)

// OverdueStatus — производное отображаемое состояние документа
// со статусом StatusCreated старше 24 часов.
const OverdueStatus = "Просрочен"

// ValidStatus проверяет принадлежность статуса фиксированному списку.
// Производное состояние StatusOverdue в список не входит.
func ValidStatus(status string) bool {
	switch status {
	case StatusCreated, StatusSent, StatusAccepted, StatusCancelled, StatusProcessed, StatusFinished:
		return true
	default:
		return false
	}
}