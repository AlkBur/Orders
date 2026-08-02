package entity

// FieldName идентифицирует поле по его GoName. Используется вне пакета
// entity для передачи набора отображаемых полей списка в Store.
type FieldName string

// Names возвращает FieldName для каждого поля.
func Names(fields []Field) []FieldName {
	names := make([]FieldName, len(fields))
	for i, f := range fields {
		names[i] = FieldName(f.GoName)
	}
	return names
}
