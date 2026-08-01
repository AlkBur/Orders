package ui

// FieldType — тип поля формы.
type FieldType int

const (
	FieldText     FieldType = 0
	FieldNumber   FieldType = 1
	FieldSelect   FieldType = 2
	FieldCheckbox FieldType = 3
	FieldPassword FieldType = 4
)

func (t FieldType) String() string {
	switch t {
	case FieldNumber:
		return "number"
	case FieldSelect:
		return "select"
	case FieldCheckbox:
		return "checkbox"
	case FieldPassword:
		return "password"
	default:
		return "text"
	}
}

// SelectOption — элемент списка выбора (FieldSelect).
// SelectOption описывает только реальные данные; placeholder для
// пустого значения — свойство Field, а не искусственная опция.
type SelectOption struct {
	Value    string
	Label    string
	Disabled bool
}

// Field — модель поля формы (Form component).
type Field struct {
	Name         string
	Label        string
	Type         FieldType
	Value        string
	Readonly     bool
	Required     bool
	Autofocus    bool
	Autocomplete string
	Icon         string
	Placeholder  string
	Options      []SelectOption
}
