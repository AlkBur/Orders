package ui

import (
	"html"
	"html/template"
	"sort"
	"strings"
)

// MarkMatches подсвечивает найденные слова внутри обычного текста.
//
// Хелпер серверной подсветки поиска. text и words — всегда обычные
// строки: никакой готовый HTML в хелпер не передаётся. Весь исходный
// текст проходит html.EscapeString, тег <mark> добавляется только вокруг
// найденных диапазонов, поэтому возвращаемый template.HTML не может
// нести пользовательский/БД-HTML (защита от XSS).
//
// Совпадения ищутся регистронезависимо (strings.ToLower) — той же
// нормализацией, что и в SQL-поиске search_normalize. Перекрывающиеся
// вхождения объединяются, чтобы не порождать вложенные <mark> и
// некорректный HTML.
//
// Пустой список words (или отсутствие совпадений) возвращает безопасно
// экранированный исходный текст без подсветки.
func MarkMatches(text string, words []string) template.HTML {
	if text == "" {
		return ""
	}
	escaped := html.EscapeString(text)
	if len(words) == 0 {
		return template.HTML(escaped)
	}

	// При некоторых Unicode-символах ToLower меняет длину строки, и тогда
	// байтовые смещения исходного текста перестают соответствовать
	// смещениям в нормализованной копии. В таком случае диапазоны не
	// гарантированы — безопаснее вернуть текст без подсветки.
	lower := strings.ToLower(text)
	if len(lower) != len(text) {
		return template.HTML(escaped)
	}

	type span struct{ start, end int }
	var spans []span
	for _, raw := range words {
		w := strings.ToLower(strings.TrimSpace(raw))
		if w == "" {
			continue
		}
		// Все вхождения слова подряд, без перекрытия внутри одного слова.
		for i := 0; ; {
			idx := strings.Index(lower[i:], w)
			if idx < 0 {
				break
			}
			start := i + idx
			end := start + len(w)
			spans = append(spans, span{start: start, end: end})
			i = end
		}
	}

	if len(spans) == 0 {
		return template.HTML(escaped)
	}

	sort.Slice(spans, func(a, b int) bool {
		if spans[a].start != spans[b].start {
			return spans[a].start < spans[b].start
		}
		return spans[a].end < spans[b].end
	})

	// Объединение перекрывающихся диапазонов.
	merged := []span{spans[0]}
	for _, s := range spans[1:] {
		last := &merged[len(merged)-1]
		if s.start <= last.end {
			if s.end > last.end {
				last.end = s.end
			}
			continue
		}
		merged = append(merged, s)
	}

	var b strings.Builder
	pos := 0
	for _, s := range merged {
		if s.start > pos {
			b.WriteString(html.EscapeString(text[pos:s.start]))
		}
		b.WriteString("<mark>")
		b.WriteString(html.EscapeString(text[s.start:s.end]))
		b.WriteString("</mark>")
		pos = s.end
	}
	if pos < len(text) {
		b.WriteString(html.EscapeString(text[pos:]))
	}

	return template.HTML(b.String())
}