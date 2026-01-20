# Examples

This directory contains executable examples demonstrating various use cases of loki-logger-go.

## Setup

For examples that connect to Loki, set the following environment variables:

```bash
export LOKI_HOST="http://localhost:3100"
export LOKI_USERNAME=""  # Optional, leave empty if no auth
export LOKI_PASSWORD=""  # Optional, leave empty if no auth
```

## Running the Examples

Each example is a standalone Go program that can be run with:

```bash
cd examples/<example-name>
go run main.go
```

## Available Examples

### 1. Basic Example (`examples/basic`)

Demonstrates basic usage with both console and Loki transports.

```bash
cd examples/basic
go run main.go
```

**Features:**
- Logging at different levels (Debug, Info, Warn, Error)
- Structured fields
- Connection to Loki server

**Prerequisites:** Loki running at `http://localhost:3100`

### 2. Console-Only Example (`examples/console_only`)

Shows how to use the logger without Loki (development mode).

```bash
cd examples/console_only
go run main.go
```

**Features:**
- No Loki connection required
- Perfect for local development
- Colored console output

**Prerequisites:** None

### 3. Labels Example (`examples/with_labels`)

Advanced example showing how to use labels for organizing logs by component.

```bash
cd examples/with_labels
go run main.go
```

**Features:**
- Global labels (environment, region)
- Child loggers with component-specific labels
- Demonstrates label inheritance

**Prerequisites:** Loki running at `http://localhost:3100`

## Starting Loki Locally

To run examples that require Loki, start a local instance with Docker:

```bash
docker run -d --name=loki -p 3100:3100 grafana/loki
```

## More Examples

For more detailed examples including web applications, HTTP middleware, and error handling patterns, see the [Usage Examples documentation](../docs/examples.md).
