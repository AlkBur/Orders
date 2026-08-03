package app

type fieldError struct {
	field string // "" — общая ошибка, не привязанная к полю
	msg   string
}

// ValidationError — модель ошибок пользовательского ввода. Не является error,
// не содержит кодов, HTTP-статусов и сведений о транспорте. Строится через
// Add/AddField и после построения не изменяется.
type ValidationError struct {
	Title       string
	fieldErrors []fieldError
}

// NewValidationError создаёт пустую модель ошибок валидации.
func NewValidationError(title string) *ValidationError {
	return &ValidationError{Title: title}
}

// AddField добавляет ошибку, привязанную к полю. Для одного поля сохраняется
// первое сообщение. Возвращает e для цепочного вызова.
func (e *ValidationError) AddField(field, message string) *ValidationError {
	if field == "" {
		return e.Add(message)
	}
	for _, fe := range e.fieldErrors {
		if fe.field == field {
			return e
		}
	}
	e.fieldErrors = append(e.fieldErrors, fieldError{field: field, msg: message})
	return e
}

// Add добавляет общую ошибку, не привязанную к конкретному полю.
func (e *ValidationError) Add(message string) *ValidationError {
	e.fieldErrors = append(e.fieldErrors, fieldError{msg: message})
	return e
}

// IsEmpty сообщает, нет ли ни одной ошибки.
func (e *ValidationError) IsEmpty() bool {
	return len(e.fieldErrors) == 0
}

// PrimaryError возвращает первое сообщение или пустую строку. Удобно для
// логирования. ValidationError намеренно не реализует error.
func (e *ValidationError) PrimaryError() string {
	if len(e.fieldErrors) == 0 {
		return ""
	}
	return e.fieldErrors[0].msg
}

// Errors возвращает сообщения в порядке добавления.
func (e *ValidationError) Errors() []string {
	msgs := make([]string, 0, len(e.fieldErrors))
	for _, fe := range e.fieldErrors {
		msgs = append(msgs, fe.msg)
	}
	return msgs
}

// FieldMap возвращает сообщения, привязанные к полям. Общие ошибки (без поля)
// в map не включаются.
func (e *ValidationError) FieldMap() map[string]string {
	m := make(map[string]string)
	for _, fe := range e.fieldErrors {
		if fe.field == "" {
			continue
		}
		if _, ok := m[fe.field]; !ok {
			m[fe.field] = fe.msg
		}
	}
	return m
}

// ErrorsMap возвращает карту поле → сообщение для инлайн-отображения ошибок
// на полной странице. Общие ошибки (без поля) складываются под ключ "".
func (e *ValidationError) ErrorsMap() map[string]string {
	m := e.FieldMap()
	for _, fe := range e.fieldErrors {
		if fe.field == "" {
			if _, ok := m[""]; !ok {
				m[""] = fe.msg
			}
		}
	}
	return m
}
