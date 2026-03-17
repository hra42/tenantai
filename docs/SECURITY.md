# Security

## Authentication

The `/services` endpoints can be protected with an API key. Set `server.admin_api_key` in `config.yaml` or via the `ADMIN_API_KEY` environment variable.

When configured, all requests to `/services/**` must include:

```
Authorization: Bearer <your-api-key>
```

When `admin_api_key` is empty (the default), authentication is disabled for backward compatibility.

The `/health`, `/ready`, and `/v1/chat/completions` endpoints are not protected by admin auth. Chat completions are gated by the `X-Service-ID` header — only registered services can use them.

## Rate Limiting

An in-memory per-IP token bucket rate limiter is available. Enable it in config:

```yaml
server:
  rate_limit:
    enabled: true
    requests_per_second: 10
```

When a client exceeds the limit, the server returns `429 Too Many Requests`. Rate limit state is per-process and not shared across instances.

## CORS

CORS is configured to allow all origins (`*`). In production, consider restricting `Access-Control-Allow-Origin` to your specific frontend domains by modifying the CORS middleware.

## Service ID Validation

Service IDs are validated to contain only lowercase alphanumeric characters and hyphens (`^[a-z0-9][a-z0-9-]*[a-z0-9]$`). This prevents directory traversal attacks since service IDs are used to construct DuckDB file paths (`data/services/{service_id}.db`).

## Recommendations

- **TLS termination:** Run behind a reverse proxy (nginx, Caddy, cloud load balancer) for TLS. The application itself serves plain HTTP.
- **Network-level access control:** Restrict access to the `/services` management endpoints at the network level in addition to API key auth.
- **API key rotation:** Store the admin API key in a secrets manager. Rotate regularly.
- **Rate limiting in production:** For multi-instance deployments, use a reverse proxy or API gateway for rate limiting instead of the built-in per-process limiter.
- **Log redaction:** Structured logs do not include request/response bodies, but ensure your log pipeline does not capture sensitive headers.
