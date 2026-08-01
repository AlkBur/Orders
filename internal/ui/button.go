package ui

// ButtonStyle — стиль кнопки.
type ButtonStyle int

const (
	ButtonDefault ButtonStyle = 0
	ButtonPrimary ButtonStyle = 1
	ButtonOutline ButtonStyle = 2
	ButtonDanger  ButtonStyle = 3
)

func (s ButtonStyle) String() string {
	switch s {
	case ButtonPrimary:
		return "primary"
	case ButtonOutline:
		return "outline"
	case ButtonDanger:
		return "danger"
	default:
		return "default"
	}
}

type Button struct {
	Style ButtonStyle
	Text  string
	URL   string
	Icon  string
}

func (b Button) Class() string {
	switch b.Style {
	case ButtonPrimary:
		return "button is-primary"
	case ButtonOutline:
		return "button is-outlined"
	case ButtonDanger:
		return "button is-danger"
	default:
		return "button"
	}
}
