# AGENTS.md

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
        temp/        # test database and temp files
```

## Rules

### No `go clean`

Do not run `go clean` (with any flags: `-cache`, `-modcache`, `-testcache`)
without explicit user permission. It clears the build cache and makes
the next build extremely slow due to `modernc.org/sqlite`.

### Temporary files

All temporary files, logs, binaries, PID files, test artifacts (cookies,
dumps, etc.) must be stored exclusively inside `tools/agent/`. Do not
write to `/tmp`, `%TEMP%`, home directory, or any other location outside
the project.

### Process management

Only terminate processes that were started by the current task and whose
PID is stored in `tools/agent/run/server.json`. Never kill unknown
processes.

Before starting a new agent server, check whether the PID stored in
`tools/agent/run/server.json` is still running. If it is running, reuse
it or stop it gracefully before starting a new one. If the process is
no longer running, remove the stale `server.json`.

Never start multiple agent servers simultaneously.

### Binary naming

The agent binary is named `orders-agent` to avoid conflicting with
`tmp/server` (used by `air`):

```bash
go build -o tools/agent/build/orders-agent ./cmd/server
```

### Config selection

The agent **must** use `ORDERS_CONFIG=config.agent.json` for all automated
testing (separate from `config.json` used by `air`/`dev`):

```bash
ORDERS_CONFIG=config.agent.json tools/agent/build/orders-agent > tools/agent/logs/server.latest.log 2>&1 &
```

The agent database lives at `tools/agent/temp/test.db` and does not
touch `data/base.db` (the dev database).

---

# Testing

## Running the server for testing

When the agent starts its own server, always use
`ORDERS_CONFIG=config.agent.json` to avoid conflicting with `air` (which
runs on `:3000`):

```bash
# Ensure directories exist (created automatically, but safe to re-run)
mkdir -p tools/agent/build tools/agent/logs tools/agent/run tools/agent/temp

# Build once, reuse the binary
go build -o tools/agent/build/orders-agent ./cmd/server

# Start server
ORDERS_CONFIG=config.agent.json tools/agent/build/orders-agent > tools/agent/logs/server.latest.log 2>&1 &
```

## Cleanup after testing

Read the PID from `tools/agent/run/server.json` and send a graceful
termination signal. If the process does not respond, use the port to
identify and terminate it:

```bash
kill -9 $(lsof -ti :3001) 2>/dev/null
```

After cleanup, remove the stale `server.json`.

## Auth module verification checklist

After any changes to the auth module, run these 8 scenarios:

1. User without password → `/set-password`
2. Set password → immediately redirected to `/orders`
3. Logout
4. Re-login with new password → `/orders`
5. Wrong password → error shown, no caching
6. F5 after error → error not from cache
7. User without password cannot access protected pages directly
8. After setting password, `/set-password` is no longer accessible (redirects to `/orders`)

## Infrastructure verification checklist

9. Development server (`:3000`) and agent server (`:3001`) can run simultaneously
   using different databases (`data/base.db` vs `tools/agent/temp/test.db`)