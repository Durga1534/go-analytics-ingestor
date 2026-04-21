# Go Analytics Ingestor 🚀

A production-grade, distributed analytics event ingestion system built with Go, featuring Redis stream consumption and PostgreSQL persistence.

## Architecture

This project follows the **Standard Go Project Layout** principles:

```
.
├── cmd/
│   └── server/              # Application entrypoint
├── internal/                # Private application code (not importable)
│   ├── cache/              # Redis layer
│   ├── config/             # Configuration management
│   ├── handlers/           # HTTP request handlers
│   ├── logger/             # Logging setup
│   ├── models/             # Data types/structures
│   ├── persistence/        # Database operations
│   └── worker/             # Business logic (event processing)
├── go.mod & go.sum         # Dependency management
└── Makefile                # Build automation
```

## Key Features

- **Distributed Consumer Group**: Scales horizontally with automatic message distribution
- **Dead Letter Queue (DLQ)**: Handles poison pills and malformed events gracefully
- **Graceful Shutdown**: Flushes pending batches before termination
- **Batch Processing**: Accumulates events for efficient database writes
- **System Metrics**: Real-time visibility into performance and health
- **Clean Architecture**: Separation of concerns with clear module boundaries

## Dependencies

- **Redis**: Event streaming and consumer groups
- **PostgreSQL**: Event persistence
- **Go 1.25+**: Modern Go features

## Environment Variables

```env
REDIS_URL=redis://user:password@host:port/db
DATABASE_URL=postgres://user:password@host:port/dbname?sslmode=disable
PORT=8080
```

## Getting Started

### Prerequisites

- Go 1.25 or higher
- Redis instance (e.g., Upstash)
- PostgreSQL instance

### Installation

```bash
# Clone repository
git clone <repo-url>
cd go-analytics-ingestor

# Install dependencies
go mod download
go mod tidy

# Copy environment template
cp .env.example .env

# Edit .env with your credentials
# REDIS_URL=...
# DATABASE_URL=...
```

### Running the Application

```bash
# Development (watch mode with air)
make dev

# Production build
make build
./bin/server

# One-liner
make run
```

### Running Tests

```bash
make test
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Tidy dependencies
make tidy
```

## API Endpoints

### POST /ingest

Ingest analytics events.

**Request:**
```json
{
  "id": "evt_123",
  "type": "page_view",
  "payload": "{\"user_id\":\"usr_456\",\"page\":\"/dashboard\"}",
  "timestamp": "2026-04-18T10:30:00Z"
}
```

**Response:** `202 Accepted`

### GET /metrics

Fetch system metrics.

**Response:**
```json
{
  "events_processed": 15230,
  "stream_pending": 42,
  "memory_usage_mb": 128,
  "uptime_seconds": 3600
}
```

## Project Structure Benefits

| Directory | Purpose |
|-----------|---------|
| `cmd/` | Clean entry point, easy to add new services |
| `internal/` | Private packages - Go prevents external imports |
| `cache/` | Redis operations isolated, reusable |
| `persistence/` | Database layer can be swapped easily |
| `handlers/` | HTTP logic separated, testable |
| `worker/` | Business logic decoupled from HTTP |
| `models/` | Centralized data types |
| `config/` | Configuration management |
| `logger/` | Logging setup, consistent across modules |

## Refactoring Summary

**Before:** 300+ line monolithic `main.go`

**After:** Modular structure with ~50-70 line files, each with a single responsibility.

### Files Created

1. `internal/models/models.go` - Event and SystemMetrics types
2. `internal/config/config.go` - Configuration loading
3. `internal/logger/logger.go` - Logger initialization
4. `internal/cache/redis.go` - Redis client wrapper
5. `internal/persistence/database.go` - Database operations
6. `internal/handlers/ingest.go` - POST /ingest endpoint
7. `internal/handlers/metrics.go` - GET /metrics endpoint
8. `internal/worker/worker.go` - Distributed event processing
9. `cmd/server/main.go` - Clean entry point
10. `Makefile` - Build automation

## Running the Refactored Project

```bash
# Use the new entry point
go run ./cmd/server/main.go

# Or build and run
make build
./bin/server

# Or use make directly
make run
```

## Next Steps

1. **Add Unit Tests**: Create `*_test.go` files in each package
2. **Add Integration Tests**: Test Redis + PostgreSQL interaction
3. **Add Docker**: Create `Dockerfile` and `docker-compose.yml`
4. **Add CI/CD**: GitHub Actions workflow for testing and deployment
5. **API Documentation**: Add Swagger/OpenAPI specs
6. **Error Handling**: Implement custom error types
7. **Metrics Export**: Add Prometheus metrics exposition

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/name`)
3. Commit changes (`git commit -am 'Add feature'`)
4. Push to branch (`git push origin feature/name`)
5. Create Pull Request

## License

MIT License - See LICENSE file for details
