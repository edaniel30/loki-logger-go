# CLAUDE.md - loki-logger-go Context Document

> **Note:** This document provides architectural context and development guidelines. For usage details, see:
> - [README.md](./README.md) - Quick start and features
> - [docs/configuration.md](./docs/configuration.md) - Complete configuration guide
> - [docs/examples.md](./docs/examples.md) - Usage examples and patterns
> - [docs/labels.md](./docs/labels.md) - Labels best practices

---

## 1. Project Overview

**loki-logger-go** is a production-grade logging library for Go with Grafana Loki integration.

**Tech Stack:**
- Go 1.25.0+
- Single dependency: `github.com/stretchr/testify` (tests only)
- Standard library heavy (net/http, encoding/json, sync, context)

**Quick Setup:**
```bash
go mod download
make test              # Run tests
make test-coverage     # 90% minimum enforced
```

**Entry Point:**
```go
import "github.com/edaniel30/loki-logger-go"

logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLokiHost("http://localhost:3100"),
)
defer logger.Close()

logger.Info(context.Background(), "Hello", nil)
```

---

## 2. Project Structure

```
loki-logger-go/
├── docs/                         # User documentation
│   ├── configuration.md          # All config options
│   ├── examples.md              # Usage patterns
│   └── labels.md                # Label guidelines
├── examples/                     # Executable examples
│   ├── basic/
│   ├── console_only/
│   └── with_labels/
├── internal/                     # Private implementation
│   ├── client/                  # HTTP client + buffer pool
│   ├── transport/               # Console & Loki transports
│   └── mocks/                   # Test utilities
├── types/                        # Public types (Entry, Level, Fields, Labels)
├── logger.go                     # Main Logger API
├── config.go                     # Config + functional options
├── errors.go                     # Error types
└── *_test.go                    # 94.7% coverage
```

**Package Dependencies:**
```
logger.go → transport (interface) → client → loki API
                ↓
         console/loki impl
```

**Visibility:**
- **Public:** Root package + `types/`
- **Internal:** Everything under `internal/` (not importable)

---

## 3. Architecture

### Layered Architecture

```
┌──────────────────────────────────┐
│   Public API (Logger)            │
│   - Debug/Info/Warn/Error/Fatal  │
│   - Flush/Close/WithLabels       │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│   Transport Layer (Strategy)     │
│   - ConsoleTransport (stdout)    │
│   - LokiTransport (HTTP)         │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│   Client Layer (HTTP)            │
│   - Retry + exponential backoff  │
│   - Stream grouping by labels    │
│   - Buffer pooling               │
└──────────────────────────────────┘
```

### Key Design Decisions

1. **No Global State:** All config/state in structs, fully testable
2. **Interface-Based:** Transport interface = easy mocking/extension
3. **Context-Aware:** All operations accept `context.Context`
4. **Non-Blocking Logging:** Errors via callback, not returned
5. **Immutable Config:** Cannot change after creation (thread-safe)

### Communication Patterns

**Logger → Transports (Fan-Out):**
```go
// Writes to all transports, one failure doesn't stop others
for _, t := range l.transports {
    if err := t.Write(ctx, entry); err != nil {
        errorHandler(t.Name(), err)  // Non-blocking callback
    }
}
```

**LokiTransport → Client (Buffered + Async):**
- Buffer entries until `BatchSize` reached or `FlushInterval` expires
- Background goroutine for periodic flushing
- Graceful shutdown with final flush

**Client → Loki (Retry):**
- Exponential backoff: 100ms, 200ms, 400ms, 800ms...
- Context-aware (respects cancellation)

---

## 4. Design Patterns

### 4.1 Functional Options Pattern
**Location:** `config.go`

**Why:** Backward compatible, self-documenting, optional parameters

```go
type Option func(*Config)

func WithAppName(name string) Option {
    return func(c *Config) { c.AppName = name }
}

// Usage:
loki.New(
    loki.DefaultConfig(),  // Provides defaults
    loki.WithAppName("my-app"),
    loki.WithLogLevel(types.LevelInfo),
)
```

### 4.2 Strategy Pattern
**Location:** `internal/transport/interface.go`

**Why:** Multiple logging destinations, easy to extend

```go
type Transport interface {
    Write(ctx context.Context, entries ...*types.Entry) error
    Flush(ctx context.Context) error
    Close() error
}
```

**Implementations:** ConsoleTransport (sync, colored), LokiTransport (async, batched)

### 4.3 Object Pool Pattern
**Location:** `internal/client/buffer.go`

**Why:** Reduce allocations, improve performance

```go
var bufferPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func Get() *bytes.Buffer { /* get + reset */ }
func Put(buf *bytes.Buffer) { /* return if < 256KB */ }
```

**Used for:** JSON encoding, string building

### 4.4 Error Wrapping Pattern
**Location:** `errors.go`

**Why:** Preserve context, inspectable with `errors.As()`

```go
type TransportError struct {
    Transport string
    Op        string
    Cause     error
}

func (e *TransportError) Unwrap() error { return e.Cause }
```

### 4.5 Template Method Pattern
**Location:** `logger.go`

All public log methods (`Debug`, `Info`, etc.) call private `log()` method:
```go
func (l *Logger) Error(ctx, msg, fields) { l.log(ctx, LevelError, msg, fields) }

func (l *Logger) log(...) {
    // 1. Level filtering
    // 2. Stack trace (Error/Fatal)
    // 3. Label construction
    // 4. Entry creation
    // 5. Transport fan-out
}
```

---

## 5. Coding Conventions

### Naming

**Files:** `logger.go`, `client.go`, `*_test.go`
**Types:** PascalCase (public), camelCase (private)
**Functions:** PascalCase (public), camelCase (private)
**Factories:** `New*` prefix
**Options:** `With*` prefix
**Acronyms:** Keep uppercase: `HTTP`, `URL`, `ID`, `JSON`, `MS` (not `Ms`)

### Code Organization

Standard file structure:
1. Package declaration
2. Imports (stdlib → external → local)
3. Constants
4. Types
5. Constructors
6. Public methods (alphabetical)
7. Private methods (alphabetical)

### Import Ordering

Enforced by `goimports` with `local-prefixes: github.com/edaniel30/loki-logger-go`

### Documentation

- **All exported symbols:** Godoc comment starting with symbol name
- **Inline comments:** Only for complex logic
- **No:** Obvious comments, TODOs in production, commented-out code

### Linting

29 enabled linters in `.golangci.yml`:
- `errcheck`, `staticcheck`, `govet`, `gocritic`, `gofmt`, `goimports`
- Coverage: 90% minimum enforced by Makefile

---

## 6. Error Handling

### Error Types

**ConfigError** - Configuration validation (returned from `New()`)
**TransportError** - Transport failures (passed to `ErrorHandler`)
**ClientError** - HTTP failures (wrapped in TransportError)

All implement `Error()` and `Unwrap()` for error chains.

### Error Propagation

```
Write() errors → ErrorHandler callback (non-blocking)
Flush() errors → ErrorHandler + return first error
Close() errors → ErrorHandler + return flush error first, then close error
```

**Key Rule:** One transport failure doesn't stop others

### Best Practices

**Do:**
- Set `ErrorHandler` in production
- Use `errors.As()` for type inspection
- Log to separate monitoring system

**Don't:**
- Panic in ErrorHandler
- Block in ErrorHandler
- Log using same logger in ErrorHandler (recursion risk)

---

## 7. Data Flow

### Log Entry Lifecycle

```
Application
    ↓
logger.Info(ctx, "msg", fields)
    ↓
1. Level filtering (skip if below threshold)
2. Stack trace (Error/Fatal only, if enabled)
3. Label construction (app, level, custom)
4. Entry creation (timestamp, fields, labels)
5. Transport fan-out (console + loki)
    ↓
ConsoleTransport: immediate stdout write
LokiTransport: buffer → batch → HTTP client
    ↓
Client: group by labels → retry → Loki API
```

### Key Data Types

**Fields** (`map[string]any`) - High-cardinality structured data
**Labels** (`map[string]string`) - Low-cardinality indexed metadata
**Entry** - Complete log entry with level, message, fields, timestamp, labels

**Transformation:**
```
Entry → Stream grouping (by labels) → PushRequest → JSON → HTTP POST
```

---

## 8. Testing

### Strategy

- **Coverage:** 94.7% (90% minimum, enforced by Makefile)
- **Pattern:** Table-driven tests for validation
- **Mocking:** `MockTransport` + `httptest` for HTTP
- **Race Detection:** `make test-race`

### Key Patterns

**Helper Functions:**
```go
newTestLogger(t) → Logger with test config
newTestLoggerWithMock(t) → Logger + MockTransport
```

**Table-Driven:**
```go
tests := []struct{ name, modify, errorField }{ /* ... */ }
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { /* ... */ })
}
```

**Excluded from Coverage:**
- `/internal/mocks` - Test utilities
- `/examples` - Example apps

---

## 9. Important Rules

### Label Cardinality (CRITICAL)

**Rule:** Labels must have **low cardinality** (< 50 unique values per label)

**Good Labels:** app, environment, region, service, version
**Bad Labels:** user_id, request_id, timestamp, ip_address, session_id

**Why:** Loki creates separate stream for each label combination. High cardinality = poor performance.

**Use Fields instead:**
```go
// ✅ Correct
logger.Info(ctx, "User login", loki.Fields{
    "user_id": 12345,        // Field (high cardinality OK)
})

// ❌ Wrong
logger.WithLabels(loki.Labels{
    "user_id": "12345",      // Label (will kill Loki performance)
})
```

**See:** [docs/labels.md](./docs/labels.md) for complete guide

### Thread Safety

**Logger is thread-safe** - Uses `sync.RWMutex` to protect transports
**Usage:** Multiple goroutines can log concurrently (no synchronization needed)

### Resource Management

**Always call `Close()`:**
```go
logger, _ := loki.New(loki.DefaultConfig(), loki.WithAppName("app"))
defer logger.Close()  // Flushes buffers, stops goroutines
```

**What Close() does:**
1. Signal background flusher to stop
2. Wait for final flush (with timeout)
3. Close all transports
4. Return error if timeout exceeded

### Context Usage

**All operations accept `context.Context`:**
- Cancellation support
- Timeout propagation
- Request-scoped values

**Never pass `nil`** - Use `context.Background()` instead

### Configuration Immutability

**Config cannot be modified after logger creation**

To change config: Create new logger
To add labels: Use `WithLabels()` (creates child logger)

### ErrorHandler Rules

**Must not:**
- Block (called during logging, will slow app)
- Panic (will crash app)
- Log recursively (use same logger)

**Should:**
- Be fast (< 1ms)
- Send to separate monitoring
- Use non-blocking channels

---

## 10. Common Commands

**Tests:**
```bash
make test                  # All tests
make test-coverage         # With 90% enforcement
make test-coverage-html    # HTML report
make test-race            # Race detection
```

**Linting:**
```bash
golangci-lint run         # All linters
golangci-lint run --fix   # Auto-fix issues
```

**Examples:**
```bash
cd examples/basic
export LOKI_HOST="http://localhost:3100"
go run main.go
```

**Loki (Docker):**
```bash
docker run -d --name=loki -p 3100:3100 grafana/loki
```

**Coverage for specific file:**
```bash
go test -coverprofile=coverage.out ./
go tool cover -func=coverage.out | grep logger.go
```

---

## 11. Development Guidelines

### Adding New Transport

1. Implement `Transport` interface in `internal/transport/`
2. Add config option in `config.go` (e.g., `WithOutputFile()`)
3. Register in `logger.setupTransports()`
4. Add tests with 90%+ coverage
5. Document in `docs/configuration.md`

**See Section 4.2** for Transport interface definition

### Adding New Functional Option

1. Add field to `Config` struct
2. Create `With*` function returning `Option`
3. Update `DefaultConfig()` if needed
4. Implement behavior in logger
5. Add tests
6. Document in `docs/configuration.md`

### Adding New Error Type

1. Define struct with relevant fields
2. Implement `Error() string` method
3. Optionally implement `Unwrap() error` for chains
4. Add constructor helper
5. Add tests (error message, `errors.As()` inspection)

### Checklist for New Features

**Must Have:**
- [ ] Follows existing architecture (layered, interface-based)
- [ ] Tests with 90%+ coverage
- [ ] Thread-safe (if applicable)
- [ ] Context-aware (if applicable)
- [ ] Godoc comments on all public symbols
- [ ] Passes `golangci-lint run`
- [ ] Backward compatible (or documented breaking change)

**Nice to Have:**
- [ ] Examples in `examples/`
- [ ] Documentation in `docs/`
- [ ] Usage example in godoc

### Code Review Criteria

**Critical:**
- No global state
- Proper error handling (all errors checked)
- Thread-safety where needed
- Tests for happy path + error cases
- 90% coverage maintained

**Important:**
- Clear naming
- Functions < 50 lines
- No deep nesting (max 3-4 levels)
- Godoc on exported symbols
- No TODOs in production code

---

## 12. Key Files Reference

**Public API:**
- `logger.go` - Main Logger (Debug/Info/Warn/Error/Fatal, Flush, Close, WithLabels)
- `config.go` - Config struct, DefaultConfig(), 13 functional options
- `errors.go` - ConfigError, TransportError, ClientError

**Internal Implementation:**
- `internal/transport/loki.go` - Batching, background flushing, graceful shutdown
- `internal/client/client.go` - HTTP client, retry logic, stream grouping
- `internal/client/buffer.go` - Buffer pool (256KB max, sync.Pool)

**Types:**
- `types/entry.go` - Entry, Fields, Labels
- `types/levels.go` - Level enum (Debug, Info, Warn, Error, Fatal)

**Testing:**
- `internal/mocks/transport.go` - MockTransport for testing
- `*_test.go` - 94.7% coverage (90% minimum)

---

## 13. Performance Notes

**Optimizations:**
- Buffer pooling (reduces GC pressure)
- Batching (reduces HTTP requests)
- Async flushing (doesn't block logging)
- Efficient JSON encoding (streams to pooled buffer)
- RWMutex (read-heavy workload optimization)

**Configuration Impact:**
- `BatchSize`: Higher = better throughput, higher latency
- `FlushInterval`: Shorter = lower latency, more HTTP requests
- `MaxRetries`: Balance reliability vs latency

**Typical Performance:**
- Console logging: ~1-5 µs per log (immediate write)
- Loki logging: ~0.5-1 µs per log (buffered) + periodic batch flush

---

## 14. External Dependencies

**Grafana Loki:**
- API: `POST /loki/api/v1/push`
- Auth: Basic auth (optional)
- Format: JSON with streams grouped by labels

**Abstraction:** Logger → Transport → Client → Loki
**Benefit:** Can swap Loki for other systems by changing transport

**See:** [docs/examples.md](./docs/examples.md) for integration patterns

---

## Quick Reference

**Most Important Files:**
1. `logger.go` - Public API (start here)
2. `config.go` - Configuration options
3. `docs/configuration.md` - Complete config guide
4. `docs/examples.md` - Usage patterns
5. `internal/transport/loki.go` - Core batching/flushing logic

**Most Important Rules:**
1. **Low cardinality labels** (see `docs/labels.md`)
2. **Always close logger** (`defer logger.Close()`)
3. **Use ErrorHandler** in production
4. **90% test coverage** minimum
5. **Thread-safe** by default

**Common Pitfalls:**
- Using high-cardinality values as labels (user IDs, timestamps)
- Forgetting to call `Close()` (loses buffered logs)
- Blocking in ErrorHandler (slows down logging)
- Not checking coverage before committing
- Modifying Config after logger creation (use WithLabels instead)

---

**For detailed usage examples and patterns, see:**
- [README.md](./README.md) - Quick start
- [docs/configuration.md](./docs/configuration.md) - All options
- [docs/examples.md](./docs/examples.md) - Complete examples
- [docs/labels.md](./docs/labels.md) - Label guidelines
