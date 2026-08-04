# AGENTS.md

AGENTS.md определяет правила поведения агента.

Если правило отсутствует в AGENTS.md, агент не должен
придумывать его самостоятельно.

Процедуры разработки, тестирования и отчётности находятся
в WORKFLOW.md.

---

# Orders

## Purpose

Orders is a lightweight server-side rendered (SSR) web application written in Go.

The project follows a minimalistic architecture:

- Go standard library whenever possible.
- SQLite as the primary database.
- HTML templates rendered on the server.
- Pico CSS as the UI framework.
- Minimal JavaScript.
- No SPA.
- No frontend build system.

---

# General Principles

1. Simplicity over abstraction.
2. Readability over cleverness.
3. Standard library first.
4. Avoid unnecessary dependencies.
5. Every abstraction must solve a real problem.

---

# Coding Style

## Go

Prefer standard library.

Avoid adding third-party packages unless they provide significant value.

Keep functions small.

Prefer explicit code over generic abstractions.

Avoid reflection.

Avoid global state.

Avoid panic except during application startup.

---

# Project Structure

The project is organized by responsibility.

- app/
- users/
- sessions/
- database/
- templates/
- static/

Each package should have a single responsibility.

---

# Templates

Templates are rendered on the server.

Release:

- templates are parsed once during startup.

Debug:

- templates are reloaded on every request.

Do not introduce template inheritance libraries.

Use Go html/template only.

---

# Static Files

Static files are served from:

/static/

Current layout:

static/
    css/
    js/
    images/

Debug serves files from disk.

Release serves embedded files.

---

# UI

Use Pico CSS.

Custom styles belong to:

static/css/main.css

Keep custom CSS small.

Avoid overriding Pico unless necessary.

Desktop and Mobile are two independent presentations of one data model. Matching
the DOM structure is not a goal; the priority is usability on each device type.
The mobile version is not required to mirror the desktop structure and is designed
as its own interface for small screens.

---

# JavaScript

JavaScript is optional.

Prefer server-side rendering.

Avoid frontend frameworks.

---

# Routing

Use chi router.

Handlers should:

- validate input
- call business logic
- render template

Business logic must not be placed inside templates.

---

# Database

SQLite.

Use prepared statements when appropriate.

Keep SQL readable.

Avoid ORMs.

## Schema Builder

Tables are described declaratively in Go using the Schema Builder,
not SQL files. Each domain package defines its own `Table`:

```go
var Table = database.Must(database.NewTable("users",
    database.Int("id").PrimaryKey().AutoIncrement(),
    database.String("login").NotNull().Unique(),
    database.String("password_hash").NotNull().Default(""),
    database.Bool("is_admin").NotNull().Default(false),
    database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
    database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
))
```

Column types are logical (string, int, bool, datetime), not SQL-specific.
The `CreateSQL()` method maps them to SQLite types at generation time.

### Rules

- `Table` is the single source of truth for the target schema.
- Business structs (`User`, `Organization`) must NOT know about the database.
- No struct tags, no init(), no global state, no ORM.
- `NewTable()` validates the schema at startup (panic via `Must()` on error).
- New databases are created directly from `Table` descriptions
  (no migration replay).

## Migrations

All migrations live in `internal/database/migrations.go` as a single list:

```go
var _ = RegisterMigrations // ensure import

func RegisterMigrations(s *database.Schema) {
    s.AddMigration(database.Migration{
        Version: 2,
        Name:    "Add column x to items",
        Up: func(ctx context.Context, tx *sql.Tx) error {
            _, err := tx.ExecContext(ctx, "ALTER TABLE items ADD COLUMN x INTEGER")
            return err
        },
    })
}
```

### Rules

- Migrations are append-only. Never edit or delete existing entries.
- Never change version numbers.
- Never insert a migration between existing ones.
- Version 1 is always the initial schema created by the Builder.
  New migrations start at version 2.
- Each migration runs in its own transaction.
- If a migration fails, the transaction rolls back and version does not advance.
- Gaps or duplicate versions cause a startup error.

## Startup Flow

```
app.New():
    1. database.NewSchema()
    2. schema.Register(domain.Table) for each domain
    3. database.OpenPath()          — opens SQLite file
    4. schema.RunMigrations(db)     — creates or updates schema

RunMigrations:
    ├─ fresh DB (version 0)     → CREATE TABLE from Table descriptions
    ├─ old system (version 1-4)  → one-time transition
    ├─ up to date                → nothing
    ├─ behind                    → apply pending migrations in order
    └─ DB ahead of code          → error
```

---

# Authentication

Authentication uses server-side sessions.

Never store passwords in plain text.

Passwords must be hashed.

---

# Error Handling

Always return errors.

Do not silently ignore errors.

Startup errors should stop the application.

Runtime errors should return proper HTTP responses.

---

# Dependencies

Before adding any dependency ask:

Can this be implemented with the standard library?

If yes, prefer the standard library.

---

# Performance

Optimize only after correctness.

Release mode should cache:

- templates
- embedded assets

Debug mode should prioritize developer convenience.

---

# Commits

Each commit should represent one logical change.

Avoid mixing unrelated changes.

Commit messages should be concise.

---

# When Modifying Code

Do not perform unrelated refactoring.

Preserve project architecture.

Preserve naming conventions.

Prefer incremental changes over rewrites.

When proposing changes, explain why they fit the existing architecture.

---

# Project Documentation

Before making architectural changes, consult the project documentation in `docs/`.

Priority:

1. `ARCHITECTURE.md` — overall architecture and design principles.
2. `FUNCTIONAL_ORDERS.md` — business rules and functional requirements.
3. `DATABASE.md` — database schema and storage rules.
4. `UI.md` — UI architecture and rendering rules.
5. `API.md` — external API contracts.

When implementation conflicts with the documentation, ask for clarification
instead of making assumptions.

---

# Agent Workspace

All temporary files must remain inside the project.

```
tools/
    agent/
        build/       # compiled binaries
        logs/        # server logs
        run/         # server.json (PID, port, started)
        temp/        # test database, cookies, temp files
        reports/     # agent reports
```

## Rules

### Go Build Cache

Go Build Cache is part of the normal development workflow.

Assume it is valid.

Do not invalidate it for benchmarking, profiling, verification,
or troubleshooting.

If you believe a clean build or cache invalidation is required,
stop and explain why. Wait for explicit user approval before
running any `go clean` command.


### No `go clean`

Never run `go clean` with any flags (`go clean`, `go clean -cache`,
`go clean -testcache`, `go clean -modcache`) unless the user explicitly
requests cache cleaning.

Do not use cache cleaning as part of benchmarking, profiling,
troubleshooting, or verification.

Go Build Cache is part of the normal development workflow.
Destroying it forces a full recompilation of dependencies
(including `modernc.org/sqlite`), which can take several minutes
and consume significant memory.

If you believe cache cleaning is necessary, stop and explain why.
Wait for explicit user approval before running any `go clean` command.

Never run `go clean` automatically.

The project must be built and tested using the existing Go build cache.
Assume the cache is valid unless the user explicitly requests a clean build.

### Temporary files

Запрещено использовать:

- `/tmp`
- `/var/tmp`
- домашнюю директорию
- любые пути вне проекта

Агент **не выбирает** пути самостоятельно.

Все временные файлы создаются строго в:

`tools/agent/temp/`

Каталог создаётся автоматически. После завершения тестирования
очищается (через `make agent-clean`).

Это правило обязательно, даже если ОС предоставляет `/tmp`.

### Process management

Only terminate processes that were started by the current task and whose
PID is stored in `tools/agent/run/server.json`. Never kill unknown
processes.

Never search for or terminate processes using `pkill`, `killall`,
or pattern-based commands (`kill $(pgrep ...)`, `ps | grep | kill`, etc.).

Only terminate the process whose PID is stored in
`tools/agent/run/server.json`.

Before starting a new agent server, check whether the PID stored in
`tools/agent/run/server.json` is still running. If it is running, reuse
it or stop it gracefully before starting a new one. If the process is
no longer running, remove the stale `server.json`.

Never start multiple agent servers simultaneously.

### Binary naming

The agent binary is named `orders-agent` to avoid conflicting with
`tmp/server` (used by `air`):

```bash
make build-agent
```

### Config selection

The agent **must** use `ORDERS_CONFIG=config.agent.json` for all automated
testing (separate from `config.json` used by `air`/`dev`):

```bash
make run-agent
```

The agent database lives at `tools/agent/temp/test.db` and does not
touch `data/base.db` (the dev database).

### Use Makefile

Если для операции существует цель в Makefile — использовать её.

Если необходимой цели нет, агент может использовать
соответствующий инструмент Go, но обязан предложить
добавить цель в Makefile.

---

# Never Assume

Если поведение не определено документацией — не придумывать.

Остановиться. Сообщить. Запросить решение.
