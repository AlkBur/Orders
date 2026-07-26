package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

// LoadRows выполняет SQL-запрос и сканирует каждую строку в срез.
// Запрос ДОЛЖЕН возвращать детерминированный порядок (используйте ORDER BY),
// иначе сравнения через AssertSlice будут непредсказуемы.
func LoadRows[T any](
	ctx context.Context,
	db *sql.DB,
	query string,
	scan func(*sql.Rows) (T, error),
	args ...any,
) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return result, nil
}

// AssertSlice проверяет, что два среза comparable-элементов совпадают.
func AssertSlice[T comparable](t *testing.T, expected, got []T) {
	t.Helper()

	if len(expected) != len(got) {
		t.Fatalf("length mismatch: expected %d rows, got %d rows\n\nexpected: %+v\ngot:      %+v",
			len(expected), len(got), expected, got)
	}

	for i := range expected {
		if expected[i] != got[i] {
			t.Fatalf("row %d mismatch:\n  expected: %+v\n  got:      %+v\n\nexpected: %+v\ngot:      %+v",
				i, expected[i], got[i], expected, got)
		}
	}
}

// AssertSliceFunc проверяет срезы с пользовательской функцией сравнения.
func AssertSliceFunc[T any](t *testing.T, expected, got []T, eq func(a, b T) bool) {
	t.Helper()

	if len(expected) != len(got) {
		t.Fatalf("length mismatch: expected %d rows, got %d rows", len(expected), len(got))
	}

	for i := range expected {
		if !eq(expected[i], got[i]) {
			t.Fatalf("row %d mismatch: expected %+v, got %+v", i, expected[i], got[i])
		}
	}
}
