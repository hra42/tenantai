# Architecture

## Multi-Tenant Isolation Model

Each registered service gets its own DuckDB file (`data/services/{service_id}.db`). The `X-Service-ID` header on every request determines which database handles the request. Services cannot access each other's data.

```
data/services/
├── default.db        ← "default" service's conversations
├── my-service.db     ← "my-service" conversations
└── analytics.db      ← "analytics" conversations
```

## Request Flow

```
HTTP Request (POST /v1/chat/completions)
    │
    ▼
Fiber Router (/v1/* group)
    │
    ▼
ServiceContext Middleware
    ├─ Extract X-Service-ID header
    ├─ Load Service from in-memory registry
    └─ Load DuckDB connection from registry
    │
    ▼
ChatHandler.HandleChatCompletion()
    ├─ Parse & validate request body
    ├─ Call orClient.ChatComplete() → OpenRouter API
    ├─ Return response to client
    └─ Async: queue conversation log (non-blocking)
            │
            ▼
      ConversationLogger (buffered channel)
            └─ Worker goroutine → INSERT into DuckDB
```

## Package Responsibilities

### `config/`
YAML config loading with `os.ExpandEnv()` for environment variable substitution. Validates required fields and applies defaults.

### `service/`
Service CRUD and lifecycle. The `ServiceManager` interface abstracts create/get/list/delete and DB connection retrieval. `DefaultServiceManager` uses an in-memory registry (map + RWMutex) for O(1) lookups. On startup, it scans the services directory to restore previously created services.

### `database/`
DuckDB connection wrapper (`OpenDB`) and idempotent schema initialization (`InitializeSchema`). Each service DB has two tables: `conversations` and `service_metadata`.

### `handler/`
HTTP handlers for three domains:
- **ChatHandler** — Chat completions (non-streaming and SSE streaming), delegates to OpenRouter via `ChatCompleter` interface
- **ServiceHandler** — Service CRUD with DB file size and conversation count in detail view
- **ConversationHandler** — Paginated conversation queries with session filtering and sorting
- **ConversationLogger** — Async writer using a buffered channel and background goroutine

### `middleware/`
- **ServiceContext** — Extracts `X-Service-ID`, loads service + DB into Fiber locals
- **ErrorHandler** — Maps `AppError`, `fiber.Error`, and sentinel errors to unified JSON error format
- **AppError** — Structured error type with HTTP status, machine-readable code, and message

### `openrouter/`
Thin wrapper around `openrouter-go` SDK. Transforms requests to SDK format, maps SDK errors to `AppError`. `ChatCompleter` interface enables mock-based testing. Debug logging in development mode.

### `models/`
Shared request/response types for chat completions, services, conversations, and errors.

## Design Decisions

### DuckDB Per Service
- **Strong isolation** — One service cannot access another's data
- **Independent scaling** — Each DB grows independently
- **Operational simplicity** — Single `.db` file per service, easy to backup/migrate
- **Trade-off** — No cross-service queries at the DB level

### Async Conversation Logging
Chat API latency should be driven by OpenRouter response time, not disk I/O. The `ConversationLogger` uses a buffered channel (256 entries) with a single worker goroutine. `Log()` is non-blocking — if the buffer is full, entries are dropped with a warning.

### In-Memory Registry
Service lookups are O(1) via a map protected by `sync.RWMutex`. The registry is rebuilt from disk on startup by scanning `.db` files and reading `service_metadata`.

### Middleware Composition
Each middleware does one thing: `ServiceContext` loads context, `ErrorHandler` maps errors. They compose naturally in Fiber's middleware chain.

## Concurrency Model

```
Main Goroutine
├─ Fiber HTTP Server
│   ├─ Request goroutine 1 → Handler → DuckDB (thread-safe sql.DB)
│   ├─ Request goroutine 2 → Handler → DuckDB
│   └─ ...
└─ ConversationLogger Worker (single goroutine)
    └─ Serialized writes to DuckDB
```

All conversation writes are serialized through a single worker goroutine to avoid write contention. Request-handling goroutines only read from DuckDB (for conversation queries) or pass data to the logger channel.

## Database Schema

**service_metadata** — Stores service name and configuration:
```sql
CREATE TABLE IF NOT EXISTS service_metadata (
    key TEXT PRIMARY KEY,
    value TEXT
);
```

**conversations** — Stores chat turns with session grouping:
```sql
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    session_id TEXT,
    model TEXT NOT NULL,
    messages JSON NOT NULL,
    finish_reason TEXT,
    metadata JSON,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```
