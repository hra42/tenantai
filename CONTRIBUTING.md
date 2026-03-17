# Contributing to tenantai

Thank you for your interest in contributing! This guide will help you get started.

## Development Setup

1. **Prerequisites**: Go 1.22+ and optionally [golangci-lint](https://golangci-lint.run/)
2. **Clone the repo**:
   ```bash
   git clone https://github.com/hra42/tenantai.git
   cd tenantai
   ```
3. **Copy and configure**:
   ```bash
   cp config.example.yaml config.yaml
   export OPENROUTER_API_KEY="your-key"
   ```
4. **Run**:
   ```bash
   go run main.go
   ```

## Running Tests

```bash
go test ./...           # All tests
go test -race ./...     # With race detection
go test -cover ./...    # With coverage
go test ./service/ -run TestServiceCreate  # Single test
```

## Code Style

- Run `golangci-lint run` before submitting
- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and small

## Submitting Changes

1. **Fork** the repository
2. **Create a branch**: `git checkout -b feature/my-change`
3. **Make your changes** with tests
4. **Run tests and lint**: `go test ./... && golangci-lint run`
5. **Commit** with a clear message describing the change
6. **Push** and open a Pull Request

## Reporting Bugs

Use the [Bug Report](https://github.com/hra42/tenantai/issues/new?template=bug_report.md) issue template. Include:

- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version)
- Relevant logs

## Requesting Features

Use the [Feature Request](https://github.com/hra42/tenantai/issues/new?template=feature_request.md) issue template. Describe:

- The problem you're trying to solve
- Your proposed solution
- Alternatives you've considered

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for a detailed overview of the system design.

## Conventions

- **Service IDs**: lowercase alphanumeric + hyphens (e.g., `my-service-1`)
- **Error responses**: unified format `{"error": {"message": "...", "code": "...", "status": N}}`
- **Async logging**: conversation logging uses a channel+worker pattern to avoid adding latency to chat responses
- **Schema creation**: always idempotent (`IF NOT EXISTS`)
- **Middleware**: composable — each middleware does one thing
