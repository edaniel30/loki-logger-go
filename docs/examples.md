# Usage Examples

This document provides practical examples of using loki-logger-go in real-world scenarios.

## Table of Contents

- [Basic Usage](#basic-usage)
- [Web Applications](#web-applications)
- [Microservices](#microservices)
- [Error Handling](#error-handling)
- [Structured Logging](#structured-logging)
- [Context and Tracing](#context-and-tracing)
- [Best Practices](#best-practices)

## Basic Usage

### Minimal Setup

```go
package main

import (
    "context"
    "log"

    "github.com/edaniel30/loki-logger-go"
)

func main() {
    // Create logger
    logger, err := loki.New(
        loki.DefaultConfig(),
        loki.WithAppName("my-service"),
        loki.WithLokiHost("http://localhost:3100"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    // Log messages
    ctx := context.Background()
    logger.Info(ctx, "Application started", nil)
    logger.Debug(ctx, "Debug information", nil)
    logger.Warn(ctx, "Warning message", nil)
    logger.Error(ctx, "Error occurred", nil)
}
```

### Console-Only Mode (Development)

```go
func main() {
    logger, err := loki.New(
        loki.DefaultConfig(),
        loki.WithAppName("dev-service"),
        loki.WithOnlyConsole(true),  // No Loki required
        loki.WithLogLevel(types.LevelDebug),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    logger.Info(context.Background(), "Running in development mode", nil)
}
```

## Web Applications

### HTTP Server with Request Logging

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/edaniel30/loki-logger-go"
)

type Server struct {
    logger *loki.Logger
}

func main() {
    // Initialize logger
    logger, _ := loki.New(
        loki.DefaultConfig(),
        loki.WithAppName("api-server"),
        loki.WithLokiHost("http://localhost:3100"),
        loki.WithLabels(types.Labels{
            "environment": "production",
            "service":     "api",
        }),
    )
    defer logger.Close()

    server := &Server{logger: logger}

    http.HandleFunc("/users", server.handleUsers)
    http.HandleFunc("/orders", server.handleOrders)

    logger.Info(context.Background(), "Server starting", types.Fields{
        "port": 8080,
    })

    http.ListenAndServe(":8080", nil)
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    requestID := r.Header.Get("X-Request-ID")

    // Create request-scoped logger
    reqLogger := s.logger.WithLabels(types.Labels{
        "handler": "users",
    })

    reqLogger.Info(r.Context(), "Request started", types.Fields{
        "request_id": requestID,
        "method":     r.Method,
        "path":       r.URL.Path,
        "user_agent": r.UserAgent(),
    })

    // Process request
    users, err := s.getUsers(r.Context(), reqLogger)
    if err != nil {
        reqLogger.Error(r.Context(), "Failed to get users", types.Fields{
            "error":      err.Error(),
            "request_id": requestID,
        })
        http.Error(w, "Internal Server Error", 500)
        return
    }

    // Log success
    duration := time.Since(start)
    reqLogger.Info(r.Context(), "Request completed", types.Fields{
        "request_id":  requestID,
        "duration_ms": duration.Milliseconds(),
        "users_count": len(users),
        "status":      200,
    })

    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "Found %d users", len(users))
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
    // Similar pattern...
}

func (s *Server) getUsers(ctx context.Context, logger *loki.Logger) ([]string, error) {
    // Simulate database query
    logger.Debug(ctx, "Querying database", types.Fields{
        "table": "users",
    })

    // Return mock data
    return []string{"user1", "user2"}, nil
}
```

### HTTP Middleware Pattern

```go
// LoggingMiddleware logs all HTTP requests
func LoggingMiddleware(logger *loki.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            requestID := r.Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = generateRequestID()
            }

            // Create request logger
            reqLogger := logger.WithLabels(types.Labels{
                "method": r.Method,
            })

            // Add logger to context
            ctx := context.WithValue(r.Context(), "logger", reqLogger)
            ctx = context.WithValue(ctx, "request_id", requestID)

            // Wrap response writer to capture status
            wrapped := &responseWriter{ResponseWriter: w, status: 200}

            reqLogger.Info(ctx, "Request started", types.Fields{
                "request_id": requestID,
                "path":       r.URL.Path,
                "remote_addr": r.RemoteAddr,
            })

            // Call next handler
            next.ServeHTTP(wrapped, r.WithContext(ctx))

            // Log completion
            duration := time.Since(start)
            reqLogger.Info(ctx, "Request completed", types.Fields{
                "request_id":  requestID,
                "status":      wrapped.status,
                "duration_ms": duration.Milliseconds(),
            })
        })
    }
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}

func generateRequestID() string {
    return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
```

## Error Handling

### Structured Error Logging

```go
func processRequest(ctx context.Context, logger *loki.Logger, data string) error {
    logger.Debug(ctx, "Processing request", types.Fields{
        "data_length": len(data),
    })

    if err := validateData(data); err != nil {
        logger.Error(ctx, "Validation failed", types.Fields{
            "error":       err.Error(),
            "data_length": len(data),
            "validator":   "schema_v2",
        })
        return fmt.Errorf("validation failed: %w", err)
    }

    if err := saveToDatabase(data); err != nil {
        logger.Error(ctx, "Database save failed", types.Fields{
            "error":      err.Error(),
            "table":      "requests",
            "operation":  "INSERT",
        })
        return fmt.Errorf("database error: %w", err)
    }

    logger.Info(ctx, "Request processed successfully", types.Fields{
        "data_length": len(data),
    })

    return nil
}
```

### Error Handler for Transport Errors

```go
func main() {
    logger, err := loki.New(
        loki.DefaultConfig(),
        loki.WithAppName("my-service"),
        loki.WithLokiHost("http://localhost:3100"),
        loki.WithErrorHandler(func(transport string, err error) {
            // Log to stderr (separate from app logs)
            fmt.Fprintf(os.Stderr, "[LOGGER ERROR] %s: %v\n", transport, err)

            // Type-safe error inspection
            var transportErr *loki.TransportError
            if errors.As(err, &transportErr) {
                fmt.Fprintf(os.Stderr, "  Transport: %s\n", transportErr.Transport)
                fmt.Fprintf(os.Stderr, "  Operation: %s\n", transportErr.Op)
                fmt.Fprintf(os.Stderr, "  Cause: %v\n", transportErr.Cause)
            }

            var clientErr *loki.ClientError
            if errors.As(err, &clientErr) {
                fmt.Fprintf(os.Stderr, "  HTTP %s %s\n", clientErr.Method, clientErr.URL)
                fmt.Fprintf(os.Stderr, "  Cause: %v\n", clientErr.Cause)
            }

            // Send to monitoring/alerting
            // metrics.IncCounter("logger_errors", map[string]string{
            //     "transport": transport,
            // })
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()
}
```

### Panic Recovery with Logging

```go
func safeProccess(ctx context.Context, logger *loki.Logger) {
    defer func() {
        if r := recover(); r != nil {
            stack := string(debug.Stack())
            logger.Fatal(ctx, "Panic recovered", types.Fields{
                "panic":       fmt.Sprintf("%v", r),
                "stack_trace": stack,
            })
        }
    }()

    // Risky operation
    doSomethingRisky()
}
```

## Best Practices

### Graceful Shutdown

```go
func main() {
    logger, err := loki.New(
        loki.DefaultConfig(),
        loki.WithAppName("my-service"),
        loki.WithLokiHost("http://localhost:3100"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Ensure cleanup on shutdown
    defer func() {
        logger.Info(context.Background(), "Application shutting down", nil)

        // Flush buffered logs
        if err := logger.Flush(); err != nil {
            fmt.Fprintf(os.Stderr, "Failed to flush logs: %v\n", err)
        }

        // Close and release resources
        if err := logger.Close(); err != nil {
            fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
        }
    }()

    // Application code
    logger.Info(context.Background(), "Application started", nil)

    // ... run application ...
}
```