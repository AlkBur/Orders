package database

import "fmt"

type ColumnType string

const (
	TypeString   ColumnType = "string"
	TypeInt      ColumnType = "int"
	TypeBool     ColumnType = "bool"
	TypeDateTime ColumnType = "datetime"
)

type Column struct {
	Name        string
	Type        ColumnType
	Description string
	IsPK        bool
	IsAutoInc   bool
	IsNotNull   bool
	IsUnique    bool
	DefaultVal  any
	RefTable    string
	RefColumn   string
	OnDeleteVal string
}

func (c Column) Comment(text string) Column {
	c.Description = text
	return c
}

func (c Column) PrimaryKey() Column {
	c.IsPK = true
	return c
}

func (c Column) AutoIncrement() Column {
	c.IsAutoInc = true
	return c
}

func (c Column) NotNull() Column {
	c.IsNotNull = true
	return c
}

func (c Column) Unique() Column {
	c.IsUnique = true
	return c
}

func (c Column) Default(v any) Column {
	c.DefaultVal = v
	return c
}

func (c Column) References(table, column string) Column {
	c.RefTable = table
	c.RefColumn = column
	return c
}

func (c Column) OnDelete(action string) Column {
	c.OnDeleteVal = action
	return c
}

func String(name string) Column {
	return Column{Name: name, Type: TypeString}
}

func Int(name string) Column {
	return Column{Name: name, Type: TypeInt}
}

func Bool(name string) Column {
	return Column{Name: name, Type: TypeBool}
}

func DateTime(name string) Column {
	return Column{Name: name, Type: TypeDateTime}
}

type Table struct {
	Name       string
	Columns    []Column
	PrimaryKey []string
}

func (t Table) SetPrimaryKey(columns ...string) Table {
	t.PrimaryKey = columns
	return t
}

func NewTable(name string, columns ...Column) (Table, error) {
	if name == "" {
		return Table{}, fmt.Errorf("table name is required")
	}

	seen := make(map[string]bool)
	pkCount := 0

	for _, col := range columns {
		if col.Name == "" {
			return Table{}, fmt.Errorf("table %q: column name is required", name)
		}
		if seen[col.Name] {
			return Table{}, fmt.Errorf("table %q: duplicate column %q", name, col.Name)
		}
		seen[col.Name] = true

		if col.IsPK {
			pkCount++
		}

		if col.IsAutoInc && (col.Type != TypeInt || !col.IsPK) {
			return Table{}, fmt.Errorf("table %q: AUTOINCREMENT requires INTEGER PRIMARY KEY", name)
		}
	}

	if pkCount > 1 {
		return Table{}, fmt.Errorf("table %q: multiple primary keys", name)
	}

	return Table{Name: name, Columns: columns}, nil
}

func Must(table Table, err error) Table {
	if err != nil {
		panic(err)
	}
	return table
}
