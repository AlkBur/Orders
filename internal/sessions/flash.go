package sessions

// FlashType — тип Flash-сообщения.
type FlashType string

const (
	FlashSuccess FlashType = "success"
	FlashError   FlashType = "error"
	FlashWarning FlashType = "warning"
	FlashInfo    FlashType = "info"
)

// Flash — сообщение, которое показывается пользователю один раз
// после редиректа (PRG). Хранится в сессии и очищается при чтении.
type Flash struct {
	Type    FlashType
	Message string
}
