# Orders

Orders is a lightweight server-side rendered (SSR) order management system written in Go.

The project focuses on simplicity, maintainability and minimal dependencies.

## Features

- Server-side rendering (SSR)
- Go standard library with minimal dependencies
- SQLite
- HTML templates
- Bulma CSS theme
- Embedded assets for release builds
- Automatic template reload in debug mode
- Session-based authentication

## Key Business Rules

- Document date and author are server-owned: a new receipt gets the current
  server date and the current session user. Client-supplied `date`/`user_id`
  values are ignored, including when copying.
- The document date always equals its creation date (`created_at`).
- Copying a document does not carry over the source date or author.
- The receipt list shows the creator (user) column and supports filtering
  by user.
- Files of a cancelled receipt remain visible but cannot be opened: the
  file endpoints return `403`.

## Requirements

- Go 1.26 or newer

## Build

### Debug

```bash
go run -tags debug ./cmd/server
```

Templates and static files are loaded directly from disk.

### Release

```bash
go build ./cmd/server
```

Templates and static files are embedded into the executable.

Config file can be overridden with the `ORDERS_CONFIG` environment variable.

## Project Structure

```
cmd/
    server/

internal/
    app/
        templates/
        static/
    database/
    ui/
    entity/
    users/
    sessions/
    customers/
    organizations/
    products/
    receipts/
```

---

## Architecture

The application follows a classic SSR architecture.

```
Browser
    │
    ▼
HTTP
    │
    ▼
chi Router
    │
    ▼
Handler
    │
    ▼
Business Logic
    │
    ▼
SQLite
```

HTML is rendered on the server.

JavaScript is optional and used only where necessary.

---

## UI

The UI is based on a Bulma theme.

Custom styles are located in:

```
internal/app/static/themes/bulma/theme.css
```

---

## Templates

### Debug

Templates are parsed on every request.

### Release

Templates are parsed once during application startup.

---

## Static Files

```
/static/
```

Debug:

- served from disk

Release:

- served from embedded filesystem

---

## Configuration

Application configuration is stored in:

```
config.json
```

---

## Documentation

Project documentation is located in `docs/`:

- `ARCHITECTURE.md` — architecture and design principles
- `FUNCTIONAL_ORDERS.md` — business rules and functional requirements
- `DATABASE.md` — database schema and storage rules
- `UI.md` — UI architecture and rendering rules
- `integration-api.md` — external integration API contracts
- `ACCEPTANCE.md` — manual acceptance checklist

Project-level rules for contributors and agents:

- `AGENTS.md`
- `WORKFLOW.md`

---

## Design Goals

- Simplicity
- Readability
- Small codebase
- Minimal dependencies
- Standard library first
- Easy deployment
- Cross-platform

---

## License

Private project.