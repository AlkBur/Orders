package app

// ResponseKind — тип ключа ответа. Типизация исключает строковые литералы
// по проекту и позволяет в будущем добавлять новые виды без рефакторинга.
type ResponseKind string

const (
	ResponseKindInfrastructure ResponseKind = "infrastructure"
)

// ResponseMessage — единственное место хранения общих пользовательских
// сообщений структурированных ответов. Компонуется в более конкретные
// ответы (ValidationResponse, InfrastructureResponse), чтобы не копировать
// поля Title и Errors между моделями.
type ResponseMessage struct {
	Title  string   `json:"title"`
	Errors []string `json:"errors"`
}

// InfrastructureResponse — структурированный ответ платформы для
// инфраструктурных ошибок (некорректный запрос, превышение лимита).
type InfrastructureResponse struct {
	Kind ResponseKind `json:"kind"`
	ResponseMessage
	// Status включается в payload для клиентов, обрабатывающих тело
	// независимо от HTTP-кода ответа. Без этого поля клиенту пришлось бы
	// полагаться исключительно на код HTTP.
	Status int `json:"status,omitempty"`
}

// NewInfrastructureResponse собирает инфраструктурный ответ.
func NewInfrastructureResponse(status int, title string, errors []string) InfrastructureResponse {
	return InfrastructureResponse{
		Kind:            ResponseKindInfrastructure,
		Status:          status,
		ResponseMessage: ResponseMessage{Title: title, Errors: errors},
	}
}
