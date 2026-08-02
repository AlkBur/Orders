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
		want  []string
	}{
		{"empty", "", nil},
		{"spaces only", "   ", nil},
		{"single word", "roma", []string{"roma"}},
		{"multiple words", "  ООО  Ромашка ", []string{"ООО", "Ромашка"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeQuery(tt.query)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("NormalizeQuery(%q) = %v, want nil", tt.query, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("NormalizeQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("NormalizeQuery(%q) = %v, want %v", tt.query, got, tt.want)
				}
			}
		})
	}
}

func TestFilterColumns(t *testing.T) {
	columns := []SearchColumn{
		{Field: "Name", SearchExpr: "c.name"},
		{Field: "OrganizationName", SearchExpr: "o.name"},
		{Field: "Unit", SearchExpr: "p.unit"},
	}

	tests := []struct {
		name   string
		fields []entity.FieldName
		want   []entity.FieldName
	}{
		{"nil fields", nil, nil},
		{"empty fields", []entity.FieldName{}, nil},
		{"subset preserves order", []entity.FieldName{"OrganizationName", "Name"}, []entity.FieldName{"Name", "OrganizationName"}},
		{"unknown field ignored", []entity.FieldName{"Name", "Bogus"}, []entity.FieldName{"Name"}},
		{"all fields", []entity.FieldName{"Name", "OrganizationName", "Unit"}, []entity.FieldName{"Name", "OrganizationName", "Unit"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterColumns(columns, tt.fields)
			if len(got) != len(tt.want) {
				t.Fatalf("FilterColumns(%v) = %v, want %v", tt.fields, got, tt.want)
			}
			for i := range got {
				if got[i].Field != tt.want[i] {
					t.Fatalf("FilterColumns(%v) = %v, want %v", tt.fields, got, tt.want)
				}
			}
		})
	}
}

func TestBuildCondition(t *testing.T) {
	cond, args := BuildCondition(SearchColumn{Field: "Name", SearchExpr: "name"}, "roma")
	if cond != `name LIKE ? ESCAPE '\'` {
		t.Fatalf("condition = %q", cond)
	}
	if len(args) != 1 || args[0] != "%roma%" {
		t.Fatalf("args = %v", args)
	}
}

func TestBuildConditionEscapesWildcards(t *testing.T) {
	_, args := BuildCondition(SearchColumn{Field: "Name", SearchExpr: "name"}, `50%_off\`)
	want := `%50\%\_off\\%`
	if args[0] != want {
		t.Fatalf("escaped word = %q, want %q", args[0], want)
	}
}

func TestBuildWhere(t *testing.T) {
	columns := []SearchColumn{
		{Field: "Name", SearchExpr: "c.name"},
		{Field: "OrganizationName", SearchExpr: "o.name"},
	}

	t.Run("nil inputs", func(t *testing.T) {
		cond, args := BuildWhere(nil, nil)
		if cond != "" || args != nil {
			t.Fatalf("BuildWhere(nil, nil) = (%q, %v)", cond, args)
		}
	})

	t.Run("no words", func(t *testing.T) {
		cond, args := BuildWhere(columns, nil)
		if cond != "" || args != nil {
			t.Fatalf("BuildWhere(columns, nil) = (%q, %v)", cond, args)
		}
	})

	t.Run("no columns", func(t *testing.T) {
		cond, args := BuildWhere(nil, []string{"roma"})
		if cond != "" || args != nil {
			t.Fatalf("BuildWhere(nil, words) = (%q, %v)", cond, args)
		}
	})

	t.Run("one word across columns", func(t *testing.T) {
		cond, args := BuildWhere(columns, []string{"roma"})
		want := `(c.name LIKE ? ESCAPE '\' OR o.name LIKE ? ESCAPE '\')`
		if cond != want {
			t.Fatalf("cond = %q, want %q", cond, want)
		}
		if len(args) != 2 {
			t.Fatalf("args = %v, want 2 args", args)
		}
	})

	t.Run("words are ANDed", func(t *testing.T) {
		cond, args := BuildWhere(columns, []string{"ооо", "ромашка"})
		want := `(c.name LIKE ? ESCAPE '\' OR o.name LIKE ? ESCAPE '\') AND (c.name LIKE ? ESCAPE '\' OR o.name LIKE ? ESCAPE '\')`
		if cond != want {
			t.Fatalf("cond = %q, want %q", cond, want)
		}
		if len(args) != 4 {
			t.Fatalf("args = %v, want 4 args", args)
		}
	})

	t.Run("words joined with spaces in query", func(t *testing.T) {
		words := NormalizeQuery(strings.Join([]string{"ооо", "ромашка"}, "  "))
		cond, _ := BuildWhere(columns, words)
		if !strings.Contains(cond, " AND ") {
			t.Fatalf("expected AND between words, got %q", cond)
		}
	})
}
