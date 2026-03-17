# Extending tenantai

## Adding Authentication (JWT Middleware)

Create a JWT middleware that runs before `ServiceContext`:

```go
// middleware/jwt.go
func JWTAuth(secret string) fiber.Handler {
    return func(c fiber.Ctx) error {
        auth := c.Get("Authorization")
        if auth == "" {
            return NewMissingHeaderError("Authorization")
        }

        token, err := jwt.Parse(strings.TrimPrefix(auth, "Bearer "),
            func(t *jwt.Token) (interface{}, error) {
                return []byte(secret), nil
            })
        if err != nil || !token.Valid {
            return &AppError{Status: 401, Code: "UNAUTHORIZED", Message: "invalid token"}
        }

        c.Locals("user_id", token.Claims.(jwt.MapClaims)["sub"])
        return c.Next()
    }
}
```

Apply to routes:

```go
v1 := app.Group("/v1", middleware.JWTAuth(cfg.JWTSecret), middleware.ServiceContext(mgr))
```

## Adding RAG (Retrieval-Augmented Generation)

1. Define a `Retriever` interface:

```go
type Retriever interface {
    Search(ctx context.Context, query string, limit int) ([]Document, error)
}
```

2. Inject into `ChatHandler` and prepend retrieved context as a system message before calling OpenRouter:

```go
docs, _ := h.retriever.Search(ctx, lastUserMessage, 5)
contextMsg := models.ChatMessage{
    Role:    "system",
    Content: fmt.Sprintf("Context:\n%s\nAnswer based on the above.", joinDocs(docs)),
}
req.Messages = append([]models.ChatMessage{contextMsg}, req.Messages...)
```

3. Implement with any vector DB (pgvector, Pinecone, Weaviate).

## Adding Cost Tracking

1. Create a `usage_logs` table in each service DB:

```sql
CREATE TABLE IF NOT EXISTS usage_logs (
    id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    estimated_cost REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

2. After a successful chat completion, log usage from `resp.Usage`:

```go
if resp.Usage != nil {
    go logUsage(db, req.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
}
```

3. Add a `GET /services/:id/usage` endpoint to query aggregated stats.

## Adding Model Routing / Fallback

Create a `ModelRouter` that selects or falls back between models:

```go
type ModelRouter interface {
    SelectModel(ctx context.Context, requested string) (string, error)
}

type FallbackRouter struct {
    Fallbacks map[string]string // e.g. "gpt-4" → "gpt-3.5-turbo"
}

func (r *FallbackRouter) SelectModel(_ context.Context, requested string) (string, error) {
    return requested, nil // Use fallback on error in retry logic
}
```

Inject into `ChatHandler`. On OpenRouter errors, retry with the fallback model:

```go
resp, err := h.orClient.ChatComplete(ctx, req)
if err != nil && h.router != nil {
    fallback, _ := h.router.SelectModel(ctx, req.Model)
    req.Model = fallback
    resp, err = h.orClient.ChatComplete(ctx, req)
}
```

## Prompt Versioning & A/B Testing

Track prompt versions per service and run A/B tests across them.

1. Create a `prompt_versions` table in each service DB:

```sql
CREATE TABLE IF NOT EXISTS prompt_versions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    system_prompt TEXT NOT NULL,
    weight REAL NOT NULL DEFAULT 1.0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

2. Define a `PromptSelector` interface:

```go
type PromptSelector interface {
    Select(ctx context.Context, db *sql.DB) (*PromptVersion, error)
}
```

3. Implement weighted random selection:

```go
type WeightedPromptSelector struct{}

func (s *WeightedPromptSelector) Select(ctx context.Context, db *sql.DB) (*PromptVersion, error) {
    rows, err := db.QueryContext(ctx,
        "SELECT id, name, system_prompt, weight FROM prompt_versions WHERE active = TRUE")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var versions []PromptVersion
    var totalWeight float64
    for rows.Next() {
        var v PromptVersion
        if err := rows.Scan(&v.ID, &v.Name, &v.SystemPrompt, &v.Weight); err != nil {
            return nil, err
        }
        versions = append(versions, v)
        totalWeight += v.Weight
    }

    // Weighted random selection
    r := rand.Float64() * totalWeight
    for _, v := range versions {
        r -= v.Weight
        if r <= 0 {
            return &v, nil
        }
    }
    return &versions[len(versions)-1], nil
}
```

4. Inject into `ChatHandler` — prepend the selected system prompt to messages and log the version ID in conversation metadata.

5. Add management endpoints:
   - `POST /services/:id/prompts` — create a prompt version
   - `GET /services/:id/prompts` — list prompt versions
   - `PUT /services/:id/prompts/:prompt_id` — update weight/active status

## Fine-Tuning Request Management

Export conversation data and manage fine-tuning jobs. Note: OpenRouter does not support fine-tuning directly — you'll need provider API keys (e.g., OpenAI) for the actual fine-tuning step.

1. Create a `fine_tune_jobs` table in each service DB:

```sql
CREATE TABLE IF NOT EXISTS fine_tune_jobs (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    base_model TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    training_file TEXT,
    result_model TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

2. Export conversations to JSONL format for training:

```go
func ExportConversationsToJSONL(db *sql.DB, w io.Writer, filter ConversationFilter) error {
    rows, err := db.Query(
        "SELECT CAST(messages AS VARCHAR) FROM conversations WHERE created_at >= ? AND created_at <= ?",
        filter.From, filter.To)
    if err != nil {
        return err
    }
    defer rows.Close()

    enc := json.NewEncoder(w)
    for rows.Next() {
        var messagesStr string
        if err := rows.Scan(&messagesStr); err != nil {
            return err
        }
        var messages map[string]interface{}
        json.Unmarshal([]byte(messagesStr), &messages)
        enc.Encode(messages)
    }
    return nil
}
```

3. Submit and poll jobs via the provider's API (e.g., OpenAI fine-tuning API), updating the `fine_tune_jobs` table with status.

4. Add endpoints:
   - `POST /services/:id/fine-tune` — create a fine-tuning job
   - `GET /services/:id/fine-tune` — list jobs
   - `GET /services/:id/fine-tune/:job_id` — get job status

## Webhook Integration

Send notifications to external services when events occur (e.g., new conversations, errors).

1. Store webhook configuration in `service_metadata`:

```sql
INSERT INTO service_metadata (key, value) VALUES
    ('webhook_url', 'https://example.com/webhook'),
    ('webhook_secret', 'whsec_...');
```

2. Create a `WebhookNotifier` using the same channel+worker pattern as `ConversationLogger`:

```go
type WebhookEvent struct {
    Type      string      `json:"type"`
    ServiceID string      `json:"service_id"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`
}

type WebhookNotifier struct {
    ch     chan WebhookEvent
    done   chan struct{}
    wg     sync.WaitGroup
    client *http.Client
}

func NewWebhookNotifier(bufferSize int) *WebhookNotifier {
    wn := &WebhookNotifier{
        ch:     make(chan WebhookEvent, bufferSize),
        done:   make(chan struct{}),
        client: &http.Client{Timeout: 10 * time.Second},
    }
    wn.wg.Add(1)
    go wn.worker()
    return wn
}
```

3. Sign payloads with HMAC-SHA256:

```go
func signPayload(payload []byte, secret string) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    return hex.EncodeToString(mac.Sum(nil))
}
```

4. Include the signature in the `X-Webhook-Signature` header when delivering.

5. Implement retry with exponential backoff (3 attempts, 1s/2s/4s delays) for failed deliveries.

## General Patterns

- **Add middleware** by composing in the Fiber route group chain
- **Add services** by implementing interfaces (`ChatCompleter`, `ServiceManager`, `Retriever`)
- **Add async work** by using the channel+worker pattern from `ConversationLogger`
- **Add endpoints** by creating new handler structs and registering routes in `main.go`
