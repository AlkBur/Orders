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

// ReceiptDeletable определяет, можно ли пометить документ на удаление
// в заданном хранимом статусе. «Просрочен» — производное состояние
// StatusCreated, поэтому отдельной проверки не требует. Пустой статус —
// legacy-данные (Создан/Отправлен без фиксированного значения),
// разрешён.
func ReceiptDeletable(status string) bool {
	switch status {
	case "", StatusCreated, StatusSent, StatusCancelled:
		return true
	default:
		return false
	}
}

// ReceiptActionable определяет, можно ли устанавливать/изменять действие
// (Удалить/Изменить) для документа в заданном статусе. Для отменённого
// документа (StatusCancelled) действие запрещено.
func ReceiptActionable(status string) bool {
	switch status {
	case StatusCancelled:
		return false
	default:
		return true
	}
}

// ReceiptFilesViewable определяет, доступны ли файлы документа для
// просмотра. Для отменённого документа файлы существуют, но открывать
// их запрещено. Пустой (legacy) статус считается доступным.
func ReceiptFilesViewable(status string) bool {
	return status != StatusCancelled
}
