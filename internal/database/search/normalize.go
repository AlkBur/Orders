package search

import "strings"

// normalizeWord — единственная реализация нормализации поискового слова.
//
// Используется только внутри пакета:
//   - NormalizeQuery — для нормализации слов запроса;
//   - SQLite-функция search_normalize (см. normalize_sqlite.go).
//
// Любое изменение алгоритма normalizeWord автоматически изменяет SQL-поиск
// через search_normalize.
func normalizeWord(s string) string {
	return strings.ToLower(s)
}
