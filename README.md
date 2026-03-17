# tenantai

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hra42/tenantai)](https://goreportcard.com/report/github.com/hra42/tenantai)

A single-binary, production-ready AI API proxy with per-tenant isolation. Not a framework — a deployable backend. Connect to 300+ models via OpenRouter, get conversation logging for free, extend only what you need.

## Features

- **Multi-tenant Isolation** — Each service gets its own DuckDB database file
- **OpenRouter Integration** — Unified interface to 300+ AI models
- **Streaming Support** — Server-Sent Events (SSE) for real-time responses
- **Async Conversation Logging** — Non-blocking logging to preserve API latency
- **Service Management API** — Create, list, and delete isolated services at runtime
- **Conversation History** — Query history with pagination and session filtering
- **Configuration** — YAML-based config with environment variable expansion

## Quick Start

```bash
git clone https://github.com/hra42/tenantai.git
cd tenantai
export OPENROUTER_API_KEY="your-api-key-here"
go run main.go
```

Verify with:

```bash
curl http://localhost:8080/health
```

### Using the Quickstart Script

The quickstart script creates a config file, sets up a default service, and starts the backend:

```bash
export OPENROUTER_API_KEY="your-api-key-here"
./scripts/quickstart.sh
```

### Docker Compose

Run the full stack (backend + frontend) with Docker Compose:

```bash
cd examples
cp .env.example .env
# Edit .env and set your OPENROUTER_API_KEY
docker compose up
```

The frontend will be available at `http://localhost:3000` and the backend at `http://localhost:8080`.

## Configuration

Config loads from `config.yaml` with `${ENV_VAR}` expansion (12-factor style).

```yaml
server:
  port: 8080
  env: development

openrouter:
  api_key: "${OPENROUTER_API_KEY}"
  base_url: "https://openrouter.ai/api/v1"

database:
  services_dir: "./data/services"
  max_connections: 10

services:
  - id: default
    name: "Default Service"
```

**Required**: `OPENROUTER_API_KEY` environment variable.

## API Overview

See [docs/API.md](docs/API.md) for full endpoint documentation with curl examples.

| Endpoint | Description |
|----------|-------------|
| `POST /v1/chat/completions` | Chat completion (requires `X-Service-ID` header) |
| `POST /services` | Create a new service |
| `GET /services` | List all services |
| `GET /services/:id` | Get service details |
| `DELETE /services/:id` | Delete a service |
| `GET /services/:id/conversations` | Query conversation history |

### Quick Example

```bash
# Create a service
curl -X POST http://localhost:8080/services \
  -H "Content-Type: application/json" \
  -d '{"id": "my-app", "name": "My App"}'

# Send a chat completion
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Service-ID: my-app" \
  -d '{
    "model": "openai/gpt-4-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed design documentation.

**Request flow**: HTTP → Fiber middleware (extract `X-Service-ID`, load DB) → Handler → OpenRouter SDK → Response + async conversation logging to DuckDB.

**Key packages**: `config/`, `service/`, `database/`, `handler/`, `middleware/`, `openrouter/`, `models/`

## Why not X?

| Feature | tenantai | LangChain | LiteLLM | OpenRouter Direct |
|---------|-----------|-----------|---------|-------------------|
| Multi-tenant isolation | Per-service DuckDB | Manual | Manual | None |
| Single binary | Yes | No (Python) | No (Python) | N/A |
| Conversation logging | Built-in, async | Manual setup | Manual setup | None |
| Model access | 300+ via OpenRouter | Multiple providers | Multiple providers | 300+ models |
| Streaming | SSE built-in | Framework-dependent | Yes | Yes |
| Complexity | Minimal, focused | High (large framework) | Moderate | API only |
| Extensibility | Middleware + interfaces | Plugins/chains | Callbacks | N/A |

**LangChain** is a comprehensive AI framework with chains, agents, and memory abstractions. If you need orchestration primitives, use LangChain. If you need a deployable backend with tenant isolation, use tenantai.

**LiteLLM** is a Python proxy that normalizes LLM APIs. It focuses on API compatibility across providers. tenantai focuses on multi-tenant isolation and conversation logging as first-class features, deployed as a single Go binary.

**OpenRouter Direct** gives you model access but no tenant isolation, conversation logging, or service management. tenantai wraps OpenRouter and adds the infrastructure layer.

## Build & Test

```bash
go build -o tenantai .      # Build
go test ./...                  # Test
go test -race ./...            # Race detection
go test -cover ./...           # Coverage
golangci-lint run              # Lint
```

## Extending

See [docs/EXTENDING.md](docs/EXTENDING.md) for guides on adding JWT auth, RAG, cost tracking, model routing, prompt versioning, fine-tuning management, and webhook integration.

## License

This project is licensed under the [MIT License](LICENSE).
