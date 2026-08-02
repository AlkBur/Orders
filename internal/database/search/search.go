package search

import (
	"strings"

	"Orders/internal/entity"
)

// SearchColumn связывает отображаемое поле списка с SQL-выражением,
// которое возвращает ту же строку, которую видит пользователь.
//
// SearchExpr — SQL-выражение, готовое к использованию в LIKE (диалект
// текущей СУБД). Точное представление строки (COALESCE, CAST и т.п.)
// определяет Store, а не модуль поиска.
//
// SearchColumn неизменяем: экземпляры создаются как пакетные литералы
// в Store и никогда не модифицируются.
type SearchColumn struct {
	Field      entity.FieldName
	SearchExpr string
}

// NormalizeQuery разбивает поисковый запрос на слова. Пустой или
// пробельный запрос возвращает nil. Модуль не знает о SQL: слова
// используются только как значения для LIKE.
func NormalizeQuery(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	return strings.Fields(query)
}

// FilterColumns оставляет только те колонки, чьи поля перечислены в fields.
// Порядок исходного слайса сохраняется; порядок fields не влияет на результат.
func FilterColumns(columns []SearchColumn, fields []entity.FieldName) []SearchColumn {
	if len(fields) == 0 {
		return nil
	}
	want := make(map[entity.FieldName]struct{}, len(fields))
	for _, f := range fields {
		want[f] = struct{}{}
	}
	var out []SearchColumn
	for _, c := range columns {
		if _, ok := want[c.Field]; ok {
			out = append(out, c)
		}
	}
	return out
}

// BuildCondition возвращает SQL-условие для одной колонки и одного слова.
// Это единственная точка, где SearchExpr попадает в LIKE: смена движка
// поиска (ILIKE, FTS5, триграммные индексы) затрагивает только её.
func BuildCondition(column SearchColumn, word string) (string, []any) {
	return column.SearchExpr + ` LIKE ? ESCAPE '\'`, []any{"%" + escapeLike(word) + "%"}
}

// BuildWhere строит условие WHERE для колонок и слов.
//
// Слова соединяются через AND, колонки — через OR:
//
//	(name LIKE ? ESCAPE '\' OR unit LIKE ? ESCAPE '\')
//	AND (name LIKE ? ESCAPE '\' OR unit LIKE ? ESCAPE '\')
//
// Чистая функция: не зависит от контекста и не обращается к базе.
// Пустые колонки или слова возвращают пустую строку без аргументов.
func BuildWhere(columns []SearchColumn, words []string) (string, []any) {
	if len(columns) == 0 || len(words) == 0 {
		return "", nil
	}

	var conds []string
	var args []any
	for _, word := range words {
		var ors []string
		for _, c := range columns {
			cond, arg := BuildCondition(c, word)
			ors = append(ors, cond)
			args = append(args, arg...)
		}
		conds = append(conds, "("+strings.Join(ors, " OR ")+")")
	}

	return strings.Join(conds, " AND "), args
}

// escapeLike экранирует спецсимволы подстановки LIKE в значении.
func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}
