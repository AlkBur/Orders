# Orders — Schema Builder

## Purpose

Schema Builder — это декларативное описание структуры БД в Go-коде.
Он заменяет ручное написание SQL-схем и является единственным источником
истины о целевой структуре базы данных.

## Table/Column API

```go
// Column type builders
database.String(name)      // → TEXT
database.Int(name)         // → INTEGER
database.Bool(name)        // → INTEGER (0/1)
database.DateTime(name)    // → DATETIME

// Column constraints (chainable)
.PrimaryKey()
.AutoIncrement()           // requires Int + PrimaryKey
.NotNull()
.Unique()
.Default(value)            // bool, int, string, or "CURRENT_TIMESTAMP"
.Comment(text)             // documentation only
.References(table, col)    // foreign key
.OnDelete(action)          // "SET NULL", "CASCADE"
```

## Adding a new table

1. Create `schema.go` in the domain package:

```go
package products

import "Orders/internal/database"

var Table = database.Must(database.NewTable("products",
    database.String("uuid").PrimaryKey(),
    database.String("name").NotNull(),
    database.Bool("active").NotNull().Default(true),
    database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
    database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
))
```

2. Register in `internal/app/schema.go`:

```go
func NewSchema() *database.Schema {
    s := database.NewSchema()

    if err := s.Register(users.Table); err != nil {
        panic(err)
    }
    // ... регистрация остальных таблиц
    if err := s.Register(products.Table); err != nil {
        panic(err)
    }

    return s
}
```

That is all. A fresh database will include the table automatically.

## Adding a migration

If the table already exists (e.g. in production), add a migration:

1. Open `internal/database/migrations.go`
2. Append to the list:

```go
func RegisterMigrations(s *database.Schema) {
    s.AddMigration(database.Migration{
        Version: 2,
        Name:    "Create products table",
        Up: func(ctx context.Context, tx *sql.Tx) error {
            _, err := tx.ExecContext(ctx, `
                CREATE TABLE products (...)
            `)
            return err
        },
    })
}
```

`AddMigration()` проверяет инварианты в момент добавления:
- version > InitialSchemaVersion (1)
- отсутствие дублей
- строгая последовательность (last+1)

## Правила

| Rule | Description |
|------|-------------|
| Append-only | Never edit or delete existing migrations |
| Version 2+ | Version 1 is the initial schema from the Builder |
| One transaction | Each migration runs in its own transaction |
| Fail-safe | Failed migration rolls back, version does not advance |
| No SQL files | All schema changes are Go code |
| No init() | Registration is explicit in `app/schema.go` |
| No struct tags | Business models don't know about the database |

## Schema immutability

Schema является неизменяемым объектом после завершения построения.
Все таблицы и миграции должны быть зарегистрированы до первого
вызова `RunMigrations()`. После начала работы схема не модифицируется.

Это означает:
- Schema не потокобезопасна (и не должна быть)
- После инициализации она фактически read-only
- Не нужно думать о блокировках или mutex'ах
