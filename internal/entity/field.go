package entity

// FieldName идентифицирует поле по его GoName. Используется вне пакета
// entity для передачи набора отображаемых полей списка в Store.
type FieldName string

// Типизированные имена свойств модели. Это не доменная логика: константы
// просто предотвращают опечатки («OrganisationName») при передаче полей
// между Handler, Store и модулем поиска.
const (
	FieldNameName             FieldName = "Name"
	FieldNameOrganizationName FieldName = "OrganizationName"
	FieldNameActive           FieldName = "Active"
	FieldNameCreatedAt        FieldName = "CreatedAt"
	FieldNameLogin            FieldName = "Login"
	FieldNameEmail            FieldName = "Email"
	FieldNameNumber           FieldName = "Number"
	FieldNameCustomerName     FieldName = "CustomerName"
	FieldNameUnit             FieldName = "Unit"
)

// Names возвращает FieldName для каждого поля.
func Names(fields []Field) []FieldName {
	names := make([]FieldName, len(fields))
	for i, f := range fields {
		names[i] = FieldName(f.GoName)
	}
	return names
}
