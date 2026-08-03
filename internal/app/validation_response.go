package app

// ValidationResponse — сериализуемое представление ValidationError.
type ValidationResponse struct {
	Title  string            `json:"title"`
	Errors []string          `json:"errors"`
	Fields map[string]string `json:"fields,omitempty"` // поле → ошибка; клиент сегодня игнорирует
}

// NewValidationResponse преобразует модель ValidationError в её сериализуемое
// представление. Преобразование — ответственность транспортного слоя; модель
// о представлении ничего не знает.
func NewValidationResponse(ve ValidationError) ValidationResponse {
	return ValidationResponse{
		Title:  ve.Title,
		Errors: ve.Errors(),
		Fields: ve.FieldMap(),
	}
}
