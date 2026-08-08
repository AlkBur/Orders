package database

import (
	"fmt"
	"strings"
)

func (t Table) CreateSQL() string {
	return t.createSQL(false)
}

func (t Table) CreateSQLIfNotExists() string {
	return t.createSQL(true)
}

func (t Table) createSQL(ifNotExists bool) string {
	var b strings.Builder

	if ifNotExists {
		b.WriteString("CREATE TABLE IF NOT EXISTS ")
	} else {
		b.WriteString("CREATE TABLE ")
	}
	b.WriteString(t.Name)
	b.WriteString(" (\n")

	columnCount := len(t.Columns)
	constraintCount := len(t.PrimaryKey)
	for _, uc := range t.UniqueConstraints {
		_ = uc
		constraintCount++
	}
	hasConstraints := constraintCount > 0
	needsExtraComma := hasConstraints

	for i, col := range t.Columns {
		b.WriteString("    ")
		b.WriteString(col.Name)
		b.WriteString(" ")
		b.WriteString(sqliteType(col.Type))

		if col.IsPK && len(t.PrimaryKey) == 0 {
			b.WriteString(" PRIMARY KEY")
		}
		if col.IsAutoInc {
			b.WriteString(" AUTOINCREMENT")
		}
		if col.IsNotNull {
			b.WriteString(" NOT NULL")
		}
		if col.IsUnique {
			b.WriteString(" UNIQUE")
		}
		if col.DefaultVal != nil {
			b.WriteString(" DEFAULT ")
			writeDefault(&b, col.DefaultVal)
		}
		if col.RefTable != "" {
			b.WriteString(" REFERENCES ")
			b.WriteString(col.RefTable)
			b.WriteString("(")
			b.WriteString(col.RefColumn)
			b.WriteString(")")
			if col.OnDeleteVal != "" {
				b.WriteString(" ON DELETE ")
				b.WriteString(col.OnDeleteVal)
			}
		}

		if i < columnCount-1 || needsExtraComma {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	if len(t.PrimaryKey) > 0 {
		b.WriteString("    PRIMARY KEY (")
		for i, pk := range t.PrimaryKey {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(pk)
		}
		b.WriteString(")")

		constraintCount--
		if constraintCount > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	for _, uc := range t.UniqueConstraints {
		b.WriteString("    UNIQUE (")
		for i, col := range uc.Columns {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(col)
		}
		b.WriteString(")")

		constraintCount--
		if constraintCount > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	b.WriteString(")")

	return b.String()
}

func sqliteType(t ColumnType) string {
	switch t {
	case TypeString:
		return "TEXT"
	case TypeInt:
		return "INTEGER"
	case TypeReal:
		return "REAL"
	case TypeBool:
		return "INTEGER"
	case TypeDateTime:
		return "DATETIME"
	case TypeBlob:
		return "BLOB"
	default:
		return string(t)
	}
}

func writeDefault(b *strings.Builder, v any) {
	switch val := v.(type) {
	case bool:
		if val {
			b.WriteString("1")
		} else {
			b.WriteString("0")
		}
	case int:
		fmt.Fprintf(b, "%d", val)
	case string:
		if val == "CURRENT_TIMESTAMP" || val == "CURRENT_DATE" || val == "CURRENT_TIME" {
			b.WriteString(val)
		} else {
			b.WriteString("'")
			b.WriteString(strings.ReplaceAll(val, "'", "''"))
			b.WriteString("'")
		}
	default:
		fmt.Fprint(b, v)
	}
}
