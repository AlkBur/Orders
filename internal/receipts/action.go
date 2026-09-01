package receipts

// Действия, запрашиваемые у 1С по документу через Integration API.
// Фиксированный список: сервер сам контролирует допустимые значения.
const (
	ActionDelete = "Удалить"
	ActionChange = "Изменить"
)

// ValidAction проверяет принадлежность действия фиксированному списку.
func ValidAction(action string) bool {
	switch action {
	case ActionDelete, ActionChange:
		return true
	default:
		return false
	}
}
