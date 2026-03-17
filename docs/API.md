# API Documentation

## Base URL

```
http://localhost:8080
```

## Error Format

All errors use a unified format:

```json
{
  "error": {
    "message": "Descriptive error message",
    "code": "ERROR_CODE",
    "status": 400
  }
}
```

### Error Codes

| Code | Status | Description |
|------|--------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request or missing required fields |
| `MISSING_HEADER` | 400 | Missing required header (e.g., `X-Service-ID`) |
| `SERVICE_NOT_FOUND` | 404 | Service ID does not exist |
| `OPENROUTER_ERROR` | varies | Error from OpenRouter API |
| `RATE_LIMITED` | 429 | Rate limited by OpenRouter |
| `INTERNAL_ERROR` | 500 | Server-side error |

---

## Endpoints

### Health Check

```
GET /health
```

**Response:**
```json
{"status": "ok"}
```

---

### Chat Completion

```
POST /v1/chat/completions
```

**Required Headers:**
- `X-Service-ID` — Service identifier
- `Content-Type: application/json`

**Optional Headers:**
- `X-Session-ID` — Group conversations by session

**Request Body:**

```json
{
  "model": "openai/gpt-4-turbo",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "What is 2 + 2?"}
  ],
  "temperature": 0.7,
  "max_tokens": 100,
  "top_p": 0.95,
  "stream": false,
  "stop": []
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | Yes | OpenRouter model ID |
| `messages` | array | Yes | Chat messages (`role` + `content`) |
| `temperature` | number | No | Sampling temperature (0–2) |
| `max_tokens` | number | No | Max response tokens |
| `top_p` | number | No | Nucleus sampling (0–1) |
| `stream` | boolean | No | Enable SSE streaming |
| `stop` | array | No | Stop sequences |

**Response (non-streaming):**

```json
{
  "id": "completion-id",
  "object": "text_completion",
  "created": 1234567890,
  "model": "openai/gpt-4-turbo",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "2 + 2 equals 4."},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 10,
    "total_tokens": 35
  }
}
```

**Response (streaming):** Server-Sent Events:

```
data: {"id":"...","choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: {"id":"...","choices":[{"index":0,"delta":{"content":" world"}}]}

data: [DONE]
```

**curl:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Service-ID: my-service" \
  -d '{"model": "openai/gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello!"}]}'
```

---

### Create Service

```
POST /services
```

**Request Body:**

```json
{"id": "my-service", "name": "My AI Service"}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Lowercase alphanumeric + hyphens, max 63 chars |
| `name` | string | Yes | Human-readable name |

**Response (201):**

```json
{"id": "my-service", "name": "My AI Service", "created_at": "2024-03-17T14:30:00Z"}
```

**curl:**

```bash
curl -X POST http://localhost:8080/services \
  -H "Content-Type: application/json" \
  -d '{"id": "my-service", "name": "My AI Service"}'
```

---

### List Services

```
GET /services
```

**Response:**

```json
[
  {"id": "default", "name": "Default Service", "created_at": "2024-03-17T10:00:00Z"},
  {"id": "my-service", "name": "My AI Service", "created_at": "2024-03-17T14:30:00Z"}
]
```

**curl:**

```bash
curl http://localhost:8080/services
```

---

### Get Service Details

```
GET /services/:id
```

**Response:**

```json
{
  "id": "my-service",
  "name": "My AI Service",
  "created_at": "2024-03-17T14:30:00Z",
  "db_file_size_bytes": 65536,
  "conversation_count": 42
}
```

**curl:**

```bash
curl http://localhost:8080/services/my-service
```

---

### Delete Service

```
DELETE /services/:id
```

**Response:** 204 No Content

**curl:**

```bash
curl -X DELETE http://localhost:8080/services/my-service
```

---

### List Conversations

```
GET /services/:id/conversations
```

**Query Parameters:**

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `limit` | int | 100 | 1000 | Results per page |
| `offset` | int | 0 | — | Skip N results |
| `session_id` | string | — | — | Filter by session |
| `sort` | string | `created_at` | — | `created_at` or `updated_at` |

**Response:**

```json
{
  "data": [
    {
      "id": "uuid",
      "created_at": "2024-03-17T14:35:00Z",
      "session_id": "session-123",
      "model": "openai/gpt-4-turbo",
      "messages": {
        "request": [{"role": "user", "content": "Hello"}],
        "response": [{"index": 0, "message": {"role": "assistant", "content": "Hi!"}, "finish_reason": "stop"}]
      },
      "finish_reason": "stop"
    }
  ],
  "total": 42,
  "limit": 10,
  "offset": 0
}
```

**curl:**

```bash
# Default pagination
curl http://localhost:8080/services/my-service/conversations

# Custom pagination with session filter
curl "http://localhost:8080/services/my-service/conversations?limit=10&offset=20&session_id=session-123"

# Sort by updated_at
curl "http://localhost:8080/services/my-service/conversations?sort=updated_at"
```
