# AI Backend - Project Plan

## Project Overview

**Name:** `tenantai` (or consider: `orbiter`, `courier`, `conductor`)

**Goal:** A minimal, extensible AI backend that abstracts OpenRouter API, provides per-service DuckDB isolation, and enables developers to quickly build and deploy multi-tenant AI applications.

**Tech Stack:**
- Language: Go 1.22+
- Framework: Fiber v3
- Database: DuckDB (per-service files)
- API Client: openrouter-go SDK
- Configuration: YAML/environment variables
- Deployment: Single binary (Docker-optional)

---

## Phase 1: Foundation (Weeks 1-2)

### 1.1 Project Setup & Structure

**File Structure:**
```
tenantai/
├── main.go                 # Entry point
├── config/
│   └── config.go          # Config loading (YAML + env)
├── service/
│   ├── manager.go         # Service CRUD, DB lifecycle
│   ├── models.go          # Service struct definitions
│   └── registry.go        # In-memory service registry
├── database/
│   ├── duckdb.go          # DuckDB connection wrapper
│   └── schema.go          # Schema initialization
├── handler/
│   ├── chat.go            # POST /v1/chat/completions
│   ├── service.go         # Service endpoints
│   └── conversation.go    # GET /services/{id}/conversations
├── middleware/
│   ├── service_context.go # X-Service-ID validation + injection
│   ├── error_handler.go   # Unified error responses
│   └── logging.go         # Request/response logging
├── openrouter/
│   └── client.go          # Wrapper around openrouter-go SDK
├── models.go              # Request/response DTOs
├── docs/
│   ├── ARCHITECTURE.md    # Design decisions
│   ├── API.md             # OpenAPI spec / endpoint reference
│   └── EXTENDING.md       # How to build on top
├── examples/
│   ├── simple-frontend/   # Minimal HTML+JS demo
│   └── docker-compose.yml # Quick local setup
├── go.mod & go.sum
└── README.md
```

**Tasks:**
- [ ] Initialize Go module: `go mod init github.com/hra42/tenantai`
- [ ] Add dependencies: fiber, duckdb, openrouter-go
- [ ] Create directory structure
- [ ] Set up basic `main.go` with Fiber app initialization
- [ ] Create `.gitignore` (*.db files, .env, etc.)

### 1.2 Configuration System

**Requirements:**
- Load config from YAML file (`config.yaml`)
- Override with environment variables (12-factor)
- Defaults for all settings

**Config Structure:**
```yaml
server:
  port: 8080
  env: development  # development | production

openrouter:
  api_key: "${OPENROUTER_API_KEY}"
  base_url: "https://openrouter.ai/api/v1"

database:
  services_dir: "./data/services"  # Where to store .db files per service
  max_connections: 10              # Per service

services:
  # Pre-register services (optional)
  - id: default
    name: "Default Service"
  - id: demo-app
    name: "Demo Frontend"
```

**Tasks:**
- [ ] Create `config/config.go` with struct unmarshaling
- [ ] Implement environment variable substitution
- [ ] Load & validate config on startup
- [ ] Panic with helpful message if required fields missing

### 1.3 Service Manager & Registry

**Service Model:**
```go
type Service struct {
    ID        string
    Name      string
    CreatedAt time.Time
    Config    map[string]interface{}  // Future extensibility
}
```

**ServiceManager Interface:**
```go
type ServiceManager interface {
    Create(ctx context.Context, id, name string) (*Service, error)
    Get(ctx context.Context, id string) (*Service, error)
    List(ctx context.Context) ([]*Service, error)
    Delete(ctx context.Context, id string) error
    GetDBConnection(ctx context.Context, id string) (*sql.DB, error)
}
```

**Tasks:**
- [ ] Implement in-memory registry (map[string]*Service with RWMutex)
- [ ] Create DuckDB file per service in `services_dir/{service_id}.db`
- [ ] Initialize schema on first creation
- [ ] Load existing services on startup (scan `services_dir`)
- [ ] Implement connection pooling per service (max_connections)
- [ ] Add validation: service ID format (alphanumeric + hyphens)

### 1.4 DuckDB Schema & Initialization

**Schema for each service DB:**

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

CREATE INDEX IF NOT EXISTS idx_conversations_session_id ON conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at);
```

**Tasks:**
- [ ] Write schema as SQL string constant
- [ ] Implement `initializeSchema(db *sql.DB)` function
- [ ] Call during service creation
- [ ] Idempotent (IF NOT EXISTS)

---

## Phase 2: API & Request Handling (Weeks 2-3)

### 2.1 Middleware: Service Context Injection

**Middleware Logic:**
1. Extract `X-Service-ID` header from request
2. Validate service exists (check registry)
3. Load DB connection for service
4. Store in Fiber `ctx.Locals("service")` and `ctx.Locals("db")`
5. If service not found: return 404 with clear error message

**Tasks:**
- [ ] Implement `middleware/service_context.go`
- [ ] Register middleware globally before routes
- [ ] Add helper function: `getServiceFromContext(c *fiber.Ctx) (*Service, error)`
- [ ] Add helper function: `getDBFromContext(c *fiber.Ctx) (*sql.DB, error)`

### 2.2 Error Handling Middleware

**Unified error response format:**
```json
{
  "error": {
    "message": "Service not found",
    "code": "SERVICE_NOT_FOUND",
    "status": 404
  }
}
```

**Tasks:**
- [ ] Define error types (ServiceNotFound, InvalidRequest, OpenRouterError, etc.)
- [ ] Create error response struct
- [ ] Implement custom error handler middleware
- [ ] Map Go errors to HTTP status codes + error codes

### 2.3 Chat Completion Handler

**Endpoint:** `POST /v1/chat/completions`

**Request Body:** (pass-through to OpenRouter)
```json
{
  "model": "openai/gpt-4-turbo",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "temperature": 0.7,
  "max_tokens": 2048,
  "stream": false
}
```

**Response:** Same as OpenRouter API (or streaming)

**Flow:**
1. Extract service from context
2. Validate request (model, messages required)
3. Call OpenRouter API via openrouter-go SDK
4. Parse response
5. Log to service's conversations table (async)
6. Return response to client

**Log Schema (messages JSON):**
```json
{
  "request": {
    "model": "openai/gpt-4-turbo",
    "messages": [...],
    "temperature": 0.7
  },
  "response": {
    "id": "...",
    "choices": [{"message": {...}, "finish_reason": "stop"}],
    "usage": {"prompt_tokens": 10, "completion_tokens": 20}
  }
}
```

**Tasks:**
- [ ] Implement `handler/chat.go` with full request validation
- [ ] Wrapper around openrouter-go SDK in `openrouter/client.go`
- [ ] Streaming support (check OpenRouter SDK capabilities)
- [ ] Non-blocking conversation logging (goroutine + channel or queue)
- [ ] Generate unique conversation ID (nanoid or UUID)
- [ ] Store session_id if provided in request (X-Session-ID header)

### 2.4 Service Management Endpoints

**Endpoints:**

1. **POST /services** - Create new service
   ```json
   {
     "id": "my-app",
     "name": "My Cool App"
   }
   ```
   Response: 201 Created with Service object

2. **GET /services** - List all services
   Response: Array of Service objects

3. **GET /services/{id}** - Get service details
   Response: Service object + metadata (created_at, DB file size, conversation count)

4. **DELETE /services/{id}** - Delete service
   Response: 204 No Content (or soft-delete: mark as deleted, don't remove DB file)

**Tasks:**
- [ ] Implement `handler/service.go` with all 4 endpoints
- [ ] Add validation: service ID uniqueness, name required
- [ ] Add metrics: DB file size, conversation count (query DuckDB)
- [ ] Add admin middleware (optional config: require header for DELETE)

### 2.5 Conversation History Endpoint

**Endpoint:** `GET /services/{id}/conversations`

**Query Parameters:**
- `limit`: 100 (default), max 1000
- `offset`: 0 (default)
- `session_id`: Filter by session (optional)
- `sort`: created_at (default) or updated_at

**Response:**
```json
{
  "data": [
    {
      "id": "conv_xyz",
      "created_at": "2025-03-17T10:00:00Z",
      "session_id": "sess_abc",
      "model": "openai/gpt-4-turbo",
      "messages": {...},
      "finish_reason": "stop"
    }
  ],
  "total": 42,
  "limit": 100,
  "offset": 0
}
```

**Tasks:**
- [ ] Implement `handler/conversation.go`
- [ ] Query conversations table with pagination + filters
- [ ] Return structured response with metadata
- [ ] Add optional `detailed: true` flag to include full messages

---

## Phase 3: Integration & Testing (Week 3-4)

### 3.1 OpenRouter SDK Integration

**Tasks:**
- [ ] Review openrouter-go SDK (https://github.com/hra42/openrouter-go)
- [ ] Create thin `openrouter/client.go` wrapper
- [ ] Handle errors from OpenRouter (rate limits, invalid models, auth)
- [ ] Support streaming responses (if SDK supports)
- [ ] Add request/response logging (optional: to stderr in debug mode)
- [ ] Document required `OPENROUTER_API_KEY` env var

### 3.2 Unit Tests

**Critical Paths:**
- [ ] Service creation + DB initialization
- [ ] Middleware: service context injection (valid + invalid service IDs)
- [ ] Chat completion: valid request → logged to DB
- [ ] Error handling: malformed requests, missing service, OpenRouter errors
- [ ] Conversation queries: pagination, filters

**Tasks:**
- [ ] Set up test package structure
- [ ] Mock DuckDB (use in-memory DuckDB or fixtures)
- [ ] Mock OpenRouter API responses
- [ ] Write table-driven tests for edge cases

### 3.3 Integration Tests

**Tasks:**
- [ ] Spin up real DuckDB in-memory for tests
- [ ] Full request flow: POST /v1/chat/completions → GET /services/{id}/conversations
- [ ] Test concurrent requests to same service (no DB conflicts)
- [ ] Test multi-service isolation (conversation A doesn't leak to service B)

### 3.4 Documentation

**Tasks:**
- [ ] **README.md**: Quick start (clone → `config.yaml` + `OPENROUTER_API_KEY` → `go run main.go`)
- [ ] **docs/API.md**: Full endpoint reference with curl examples
- [ ] **docs/ARCHITECTURE.md**: Design decisions + data flow diagram
- [ ] **docs/EXTENDING.md**: "How to add auth", "How to add RAG", "How to add cost tracking"
- [ ] Add inline code comments for non-obvious logic

---

## Phase 4: Examples & Demo (Week 4)

### 4.1 Simple Frontend Example

**Location:** `examples/simple-frontend/`

**Tech:** HTML + Vanilla JS (no build step)

**Features:**
- Create service on load (or use pre-created service)
- Chat UI: message input, send button, conversation display
- Displays conversation history on load
- Shows token usage (if OpenRouter includes in response)

**Files:**
- [ ] `index.html` - Chat interface
- [ ] `app.js` - API calls + DOM updates
- [ ] `style.css` - Basic styling
- [ ] `README.md` - How to run

**Tasks:**
- [ ] Design simple but polished chat UI (consider your cyberpunk aesthetic 🎮)
- [ ] Implement API client helper functions
- [ ] Test against running backend
- [ ] Screenshot for main README

### 4.2 Docker Compose Setup

**File:** `examples/docker-compose.yml`

**Services:**
- `tenantai`: Go app + DuckDB
- `simple-frontend`: Static HTTP server

**Volumes:**
- Mount `./data/services` for DuckDB persistence
- Mount `.env` for config

**Tasks:**
- [ ] Write docker-compose.yml
- [ ] Create Dockerfile for Go app (multi-stage build)
- [ ] Document: `docker-compose up` → visit `http://localhost:3000`
- [ ] Test locally

### 4.3 Quick Start Script

**File:** `scripts/quickstart.sh`

**Actions:**
1. Check Go installed
2. Copy `config.example.yaml` → `config.yaml`
3. Prompt for `OPENROUTER_API_KEY`
4. Prompt for port
5. Run `go run main.go`

**Tasks:**
- [ ] Write shell script
- [ ] Make it cross-platform (check for bash availability)

---

## Phase 5: Production Readiness (Week 4-5)

### 5.1 Configuration & Deployment

**Tasks:**
- [ ] Support config from file + environment variables
- [ ] Add graceful shutdown (SIGTERM handler)
- [ ] Add health check endpoint: `GET /health` → `{"status": "ok"}`
- [ ] Add readiness endpoint: `GET /ready` → check all services' DBs accessible
- [ ] Logging: structured logs (JSON format) with severity levels

### 5.2 Observability

**Tasks:**
- [ ] Request logging: method, path, status, duration, service_id
- [ ] Error logging: stack traces in development, sanitized in production
- [ ] Optional: metrics endpoint `GET /metrics` (Prometheus format)
  - Requests by endpoint, by service
  - OpenRouter API latency
  - DB operation timing

### 5.3 Security

**Tasks:**
- [ ] Add optional auth for `/services` endpoints (header-based API key)
- [ ] Validate service ID format (prevent directory traversal)
- [ ] Rate limiting (optional: per service or global)
- [ ] Document security considerations in `docs/SECURITY.md`

### 5.4 Performance

**Tasks:**
- [ ] Benchmark conversation logging (ensure non-blocking)
- [ ] Profile memory usage under load
- [ ] Test connection pooling behavior
- [ ] Document scaling limits

---

## Phase 6: Launch & Community (Week 5+)

### 6.1 GitHub Setup

**Tasks:**
- [ ] Push to `github.com/hra42/tenantai`
- [ ] Create GitHub issue templates (bug, feature request, discussion)
- [ ] Add CONTRIBUTING.md
- [ ] License: MIT (or your preference)

### 6.2 OSS Positioning

**Tasks:**
- [ ] Write compelling README with the narrative we outlined
- [ ] Create badges (Go version, license, etc.)
- [ ] Add "Why not X?" section comparing to alternatives (LangChain, LiteLLM, etc.)
- [ ] Create 2-3 minute demo GIF (chat UI in action)

### 6.3 Future Extension Hooks (Document but Don't Build Yet)

**In `docs/EXTENDING.md`, outline paths for:**
- [ ] User authentication (JWT middleware example)
- [ ] RAG integration (Qdrant or PostgreSQL vector types)
- [ ] Cost tracking per user/service
- [ ] Prompt versioning + A/B testing
- [ ] Fine-tuning request management
- [ ] Model routing/fallback logic
- [ ] Webhook integration (notify on completion)

**Leave skeleton for these in code comments (e.g., "TODO: Add user context to conversation logs")**

---

## Success Criteria

### Phase 1 ✓
- Single binary builds and runs
- Configuration loads from YAML + env
- Service manager creates isolated DuckDBs

### Phase 2 ✓
- Chat completion endpoint works end-to-end
- Conversations logged to service DB
- All 4 service management endpoints working
- Conversation history queryable

### Phase 3 ✓
- Unit tests: 70%+ coverage on core logic
- Integration tests: E2E flow passes
- Documentation complete + clear

### Phase 4 ✓
- Simple frontend runs in `docker-compose up`
- Screenshots in README
- Quickstart takes <5 minutes

### Phase 5 ✓
- Graceful shutdown
- Health checks pass
- Structured logging
- Runs on production-like setup

### Phase 6 ✓
- GitHub repo public + documented
- README tells the story
- Extension hooks clear + documented

---

## Timeline Estimate

- **Weeks 1-2:** Foundation (config, service manager, DuckDB schema)
- **Weeks 2-3:** API endpoints + middleware
- **Week 3-4:** Testing + integration
- **Week 4:** Examples + demo
- **Week 5:** Production polish + launch

**Realistically:** 4-5 weeks of focused work (assuming 20-30 hrs/week)

---

## Open Decisions

1. **Streaming Support:** How much of the OpenRouter streaming API should we support? (Full HTTP chunked? WebSocket?)
2. **Soft Delete vs Hard Delete:** When deleting a service, keep the .db file archived or remove it?
3. **Service ID Constraints:** Allow only lowercase alphanumeric + hyphens? Or more permissive?
4. **Admin Auth:** Require header-based API key for `/services` endpoints, or open by default?
5. **Project Name:** Stick with `tenantai` or something more creative?

---

## Notes for Implementation

- Use your openrouter-go SDK directly; don't rewrite HTTP logic
- DuckDB files per service = maximum isolation, minimal complexity
- Keep middleware simple and composable (each middleware does one thing)
- Async logging prevents chat latency from spiking
- Leave extension points documented but not implemented (anti-bloat)
- Consider your deployment at seventhings: can this run on your existing Alpine Linux fleet?
