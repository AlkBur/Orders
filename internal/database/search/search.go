package search

import (
	"strings"

	"Orders/internal/entity"
)

// SearchQuery — результат лексического разбора поискового запроса.
//
// Неизменяемое значение с инвариантами:
//
//	Original — всегда исходная строка пользователя без изменений.
//	Words    — всегда нормализованы (normalizeWord) и никогда
//	          не содержат пустых строк.
//
// Все последующие функции полагаются на эти инварианты и не делают
// повторных проверок или нормализации.
//
// Future extensions:
//
//   - quoted phrases
//   - excluded words
//   - field:value
//   - OR groups
type SearchQuery struct {
	Original string
	Words    []string
}

// MappedColumn связывает отображаемое поле списка с SQL-выражением,
// которое возвращает ту же строку, которую видит пользователь.
//
// Field — entity.FieldName по GoName, ключ соответствия с visibleFields.
// Expression — SQL-выражение, готовое к использованию в LIKE (диалект
// текущей СУБД). Точное представление строки (COALESCE, CAST и т.п.)
// определяет Store, а не модуль поиска.
//
// MappedColumn неизменяем: экземпляры создаются как пакетные литералы
// в Store и никогда не модифицируются.
type MappedColumn struct {
	Field      entity.FieldName
	Expression string
}

// SearchColumn — колонка поиска после фильтрации по отображаемым полям.
// Содержит только Expression: BuildWhere и BuildPredicate не знают о Field.
type SearchColumn struct {
	Expression string
}

// NormalizeQuery разбивает поисковый запрос на нормализованные слова.
//
// Original сохраняет исходную строку пользователя без изменений.
// Пустой или пробельный запрос возвращает нулевой SearchQuery.
// Words не содержат пустых строк (слова разделяются strings.Fields).
func NormalizeQuery(query string) SearchQuery {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return SearchQuery{}
	}

	words := strings.Fields(trimmed)
	for i, w := range words {
		words[i] = normalizeWord(w)
	}

	return SearchQuery{Original: query, Words: words}
}

// VisibleColumns оставляет только те поисковые колонки, чьи поля
// перечислены в visible. Порядок исходного слайса сохраняется.
// Возвращает SearchColumn без Field — колонки, готовые для BuildWhere.
func VisibleColumns(columns []MappedColumn, visible []entity.FieldName) []SearchColumn {
	if len(visible) == 0 {
		return nil
	}

	want := make(map[entity.FieldName]struct{}, len(visible))
	for _, f := range visible {
		want[f] = struct{}{}
	}

	var out []SearchColumn
	for _, c := range columns {
		if _, ok := want[c.Field]; ok {
			out = append(out, SearchColumn{Expression: c.Expression})
		}
	}
	return out
}

// BuildPredicate возвращает SQL-предикат для одного слова и одной колонки.
// Это единственная точка, где Expression попадает в LIKE: смена движка
// поиска (ILIKE, FTS5, триграммные индексы) затрагивает только её.
//
// Слово уже нормализовано (инвариант SearchQuery) — здесь не нормализуется.
func BuildPredicate(column SearchColumn, word string) (string, []any) {
	return `search_normalize(` + column.Expression + `) LIKE ? ESCAPE '\'`,
		[]any{"%" + escapeLike(word) + "%"}
}

// BuildWhere строит условие WHERE для колонок и слов.
//
// Слова соединяются через AND, колонки — через OR:
//
//	(search_normalize(name) LIKE ? ESCAPE '\' OR search_normalize(unit) LIKE ? ESCAPE '\')
//	AND (search_normalize(name) LIKE ? ESCAPE '\' OR search_normalize(unit) LIKE ? ESCAPE '\')
//
// Чистая функция: не зависит от контекста и не обращается к базе.
// Пустые колонки или слова возвращают пустую строку без аргументов.
func BuildWhere(columns []SearchColumn, q SearchQuery) (string, []any) {
	if len(columns) == 0 || len(q.Words) == 0 {
		return "", nil
	}

	var conds []string
	var args []any
	for _, word := range q.Words {
		var ors []string
		for _, c := range columns {
			pred, arg := BuildPredicate(c, word)
			ors = append(ors, pred)
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
