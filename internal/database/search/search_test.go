package search

import (
	"strings"
	"testing"

	"Orders/internal/entity"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  SearchQuery
	}{
		{
			name:  "empty",
			query: "",
			want:  SearchQuery{},
		},
		{
			name:  "spaces only",
			query: "   ",
			want:  SearchQuery{},
		},
		{
			name:  "single word",
			query: "roma",
			want:  SearchQuery{Original: "roma", Words: []string{"roma"}},
		},
		{
			name:  "multiple words normalized",
			query: "  ООО  Ромашка ",
			want:  SearchQuery{Original: "  ООО  Ромашка ", Words: []string{"ооо", "ромашка"}},
		},
		{
			name:  "case insensitive",
			query: "RoMa",
			want:  SearchQuery{Original: "RoMa", Words: []string{"roma"}},
		},
		{
			name:  "trim collapse case",
			query: "   ООО      Ромашка     ",
			want:  SearchQuery{Original: "   ООО      Ромашка     ", Words: []string{"ооо", "ромашка"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeQuery(tt.query)
			if got.Original != tt.want.Original {
				t.Fatalf("NormalizeQuery(%q).Original = %q, want %q", tt.query, got.Original, tt.want.Original)
			}
			if len(got.Words) != len(tt.want.Words) {
				t.Fatalf("NormalizeQuery(%q).Words = %v, want %v", tt.query, got.Words, tt.want.Words)
			}
			for i := range got.Words {
				if got.Words[i] != tt.want.Words[i] {
					t.Fatalf("NormalizeQuery(%q).Words = %v, want %v", tt.query, got.Words, tt.want.Words)
				}
			}
		})
	}
}

func TestNormalizeQueryInvariants(t *testing.T) {
	// Слова никогда не содержат пустых строк.
	q := NormalizeQuery("ООО    Ромашка")
	for _, w := range q.Words {
		if strings.TrimSpace(w) == "" {
			t.Fatalf("word %q is empty", w)
		}
	}
}

func TestVisibleColumns(t *testing.T) {
	columns := []MappedColumn{
		{Field: entity.FieldNameName, Expression: "c.name"},
		{Field: entity.FieldNameOrganizationName, Expression: "o.name"},
		{Field: entity.FieldNameUnit, Expression: "p.unit"},
	}

	tests := []struct {
		name    string
		visible []entity.FieldName
		want    []string
	}{
		{"nil visible", nil, nil},
		{"empty visible", []entity.FieldName{}, nil},
		{
			"subset preserves order",
			[]entity.FieldName{entity.FieldNameOrganizationName, entity.FieldNameName},
			[]string{"c.name", "o.name"},
		},
		{"unknown field ignored",
			[]entity.FieldName{entity.FieldNameName, "Bogus"},
			[]string{"c.name"},
		},
		{
			"all fields",
			[]entity.FieldName{entity.FieldNameName, entity.FieldNameOrganizationName, entity.FieldNameUnit},
			[]string{"c.name", "o.name", "p.unit"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VisibleColumns(columns, tt.visible)
			if len(got) != len(tt.want) {
				t.Fatalf("VisibleColumns(%v, %v) = %v, want %v", columns, tt.visible, got, tt.want)
			}
			for i := range got {
				if got[i].Expression != tt.want[i] {
					t.Fatalf("VisibleColumns(%v, %v) = %v, want %v", columns, tt.visible, got, tt.want)
				}
			}
		})
	}
}

func TestBuildPredicate(t *testing.T) {
	cond, args := BuildPredicate(SearchColumn{Expression: "name"}, "roma")
	want := `search_normalize(name) LIKE ? ESCAPE '\'`
	if cond != want {
		t.Fatalf("condition = %q, want %q", cond, want)
	}
	if len(args) != 1 || args[0] != "%roma%" {
		t.Fatalf("args = %v", args)
	}
}

func TestBuildPredicateEscapesWildcards(t *testing.T) {
	_, args := BuildPredicate(SearchColumn{Expression: "name"}, `50%_off\`)
	want := `%50\%\_off\\%`
	if args[0] != want {
		t.Fatalf("escaped word = %q, want %q", args[0], want)
	}
}

// BuildPredicate не нормализует: ожидается, что слово уже нормализовано
// (инвариант SearchQuery). Здесь проверяем, что оно проходит как есть.
func TestBuildPredicateDoesNotNormalize(t *testing.T) {
	_, args := BuildPredicate(SearchColumn{Expression: "name"}, "RoMa")
	if args[0] != "%RoMa%" {
		t.Fatalf("args = %v, want unchanged word", args)
	}
}

func TestBuildWhere(t *testing.T) {
	columns := []SearchColumn{
		{Expression: "c.name"},
		{Expression: "o.name"},
	}

	t.Run("nil inputs", func(t *testing.T) {
		cond, args := BuildWhere(nil, SearchQuery{})
		if cond != "" || args != nil {
			t.Fatalf("BuildWhere(nil, {}) = (%q, %v)", cond, args)
		}
	})

	t.Run("no words", func(t *testing.T) {
		cond, args := BuildWhere(columns, SearchQuery{})
		if cond != "" || args != nil {
			t.Fatalf("BuildWhere(columns, {}) = (%q, %v)", cond, args)
		}
	})

	t.Run("no columns", func(t *testing.T) {
		cond, args := BuildWhere(nil, SearchQuery{Words: []string{"roma"}})
		if cond != "" || args != nil {
			t.Fatalf("BuildWhere(nil, words) = (%q, %v)", cond, args)
		}
	})

	t.Run("one word across columns", func(t *testing.T) {
		cond, args := BuildWhere(columns, SearchQuery{Words: []string{"roma"}})
		want := `(search_normalize(c.name) LIKE ? ESCAPE '\' OR search_normalize(o.name) LIKE ? ESCAPE '\')`
		if cond != want {
			t.Fatalf("cond = %q, want %q", cond, want)
		}
		if len(args) != 2 {
			t.Fatalf("args = %v, want 2 args", args)
		}
	})

	t.Run("words are ANDed", func(t *testing.T) {
		cond, _ := BuildWhere(columns, parseQuery(t, "ооо ромашка"))
		want := `(search_normalize(c.name) LIKE ? ESCAPE '\' OR search_normalize(o.name) LIKE ? ESCAPE '\') AND (search_normalize(c.name) LIKE ? ESCAPE '\' OR search_normalize(o.name) LIKE ? ESCAPE '\')`
		if cond != want {
			t.Fatalf("cond = %q, want %q", cond, want)
		}
	})

	t.Run("word order irrelevant", func(t *testing.T) {
		// Words нормализованы и AND-аны; порядок не влияет на результат.
		_, args := BuildWhere(columns, parseQuery(t, "Ромашка ООО"))
		// Каждое слово даёт аргумент на каждую колонку: 2 слова × 2 колонки.
		if len(args) != 4 {
			t.Fatalf("args = %v, want 4 args", args)
		}
	})
}

func parseQuery(t *testing.T, q string) SearchQuery {
	t.Helper()
	return NormalizeQuery(q)
}
