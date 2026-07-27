package database

import (
	"fmt"
	"strings"
)

func (t Table) CreateSQL() string {
	var b strings.Builder

	b.WriteString("CREATE TABLE ")
	b.WriteString(t.Name)
	b.WriteString(" (\n")

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

		if i < len(t.Columns)-1 || len(t.PrimaryKey) > 0 {
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
		b.WriteString(")\n")
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
	case TypeBool:
		return "INTEGER"
	case TypeDateTime:
		return "DATETIME"
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
