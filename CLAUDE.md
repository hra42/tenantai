# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tenantai is a minimal, extensible AI backend in Go that abstracts the OpenRouter API, provides per-service DuckDB isolation, and supports multi-tenant AI applications.
## Tech Stack

- **Language:** Go 1.26+
- **Framework:** Fiber v3
- **Database:** DuckDB (one `.db` file per service for tenant isolation)
- **AI API:** OpenRouter via `openrouter-go` SDK (github.com/hra42/openrouter-go)
- **Config:** YAML + environment variable overrides

## Build & Run Commands

```bash
# Run
go run main.go

# Build
go build -o tenantai .

# Test
go test ./...

# Run a single test
go test ./service/ -run TestServiceCreate

# Test with race detection
go test -race ./...

# Lint (if golangci-lint installed)
golangci-lint run
```

## Architecture

**Multi-tenant isolation model:** Each registered service gets its own DuckDB file (`data/services/{service_id}.db`). The `X-Service-ID` header on every request determines which database handles the request.

**Request flow:** HTTP request → Fiber middleware (extract `X-Service-ID`, load DB connection into context) → Handler → OpenRouter SDK → Response + async conversation logging to DuckDB.

**Key packages:**
- `config/` — YAML + env config loading
- `service/` — Service CRUD, in-memory registry (map + RWMutex), DB lifecycle management
- `database/` — DuckDB connection wrapper, schema initialization
- `handler/` — HTTP handlers: chat completions, service management, conversation history
- `middleware/` — Service context injection (`X-Service-ID`), error handling, logging
- `openrouter/` — Thin wrapper around openrouter-go SDK

**API endpoints:**
- `POST /v1/chat/completions` — Proxied to OpenRouter, logged per-service
- `POST /services`, `GET /services`, `GET /services/{id}`, `DELETE /services/{id}` — Service management
- `GET /services/{id}/conversations` — Query conversation history with pagination

## Configuration

Config loads from `config.yaml` with env var overrides (12-factor). Required: `OPENROUTER_API_KEY`.

## Conventions

- Service IDs: lowercase alphanumeric + hyphens
- Error responses use a unified format: `{"error": {"message": "...", "code": "...", "status": N}}`
- Conversation logging is non-blocking (goroutine/channel) to avoid adding latency to chat responses
- Schema creation is idempotent (`IF NOT EXISTS`)
- Middleware is composable — each middleware does one thing

## Gotchas

- **Fiber v3 closes `io.Closer` locals:** Fiber v3 automatically calls `Close()` on any value stored in `c.Locals()` that implements `io.Closer` when the request context is cleaned up. **Never store `*sql.DB` directly in locals** — it will be closed after the first request. Use `middleware.SetDBInContext()` which wraps the DB in a `dbRef` struct that doesn't implement `io.Closer`.
- **DuckDB TIMESTAMP/JSON scanning:** DuckDB returns `TIMESTAMP` columns as `time.Time` and `JSON` columns as `interface{}` (map), not strings. Use `CAST(col AS VARCHAR)` in queries when scanning into Go `string` variables.
- **OpenRouter client interface:** `openrouter.ChatCompleter` interface enables mock-based testing for handlers. `ChatHandler` takes the interface, not the concrete `*Client`.
- **Async logger lifecycle in tests:** When testing conversation logging, close the `ConversationLogger` before querying the DB to ensure writes are flushed. Close the logger *before* closing the DB/service manager.
