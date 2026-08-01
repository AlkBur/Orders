package ui

// AlertType — тип блока сообщений.
type AlertType int

const (
	AlertInfo AlertType = iota
	AlertSuccess
	AlertWarning
	AlertError
)

func (t AlertType) String() string {
	switch t {
	case AlertSuccess:
		return "success"
	case AlertWarning:
		return "warning"
	case AlertError:
		return "error"
	default:
		return "info"
	}
}

// AlertData — модель блока сообщений (Alert component).
type AlertData struct {
	Type     AlertType
	Messages []string
}
