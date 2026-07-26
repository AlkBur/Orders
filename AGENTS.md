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