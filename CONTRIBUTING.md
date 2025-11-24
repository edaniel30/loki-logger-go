# Contributing to Loki Logger Go

First off, thank you for considering contributing to Loki Logger Go! It's people like you that make this project better for everyone.

## Table of Contents

- [Getting Started](#getting-started)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Pull Requests](#pull-requests)
- [Development Setup](#development-setup)
- [Style Guidelines](#style-guidelines)
  - [Go Code Style](#go-code-style)
  - [Commit Messages](#commit-messages)
  - [Documentation](#documentation)
- [Testing](#testing)
- [Project Structure](#project-structure)

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/edaniel30/loki-logger-go.git
   cd loki-logger-go
   ```
3. **Install dependencies**:
   ```bash
   go mod download
   ```
4. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/my-new-feature
   ```

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When creating a bug report, include as many details as possible:

**Use the following template:**

```markdown
**Description:**
A clear and concise description of the bug.

**Steps to Reproduce:**
1. Go to '...'
2. Call function '...'
3. See error

**Expected Behavior:**
What you expected to happen.

**Actual Behavior:**
What actually happened.

**Environment:**
- Go version: [e.g., 1.21.0]
- OS: [e.g., macOS 14.0, Ubuntu 22.04]
- Loki version: [e.g., 2.9.0]

**Code Sample:**
```go
// Minimal code to reproduce the issue
logger, _ := loki.New(...)
logger.Info("test")
```

**Additional Context:**
Any other context about the problem.
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the suggested enhancement
- **Explain why this enhancement would be useful** to most users
- **Provide examples** of how the feature would be used
- **List alternatives** you've considered

### Pull Requests

1. **Follow the style guidelines** (see below)
2. **Write tests** for your changes
3. **Update documentation** if needed
4. **Ensure all tests pass** before submitting
5. **Reference relevant issues** in your PR description

**Pull Request Template:**

```markdown
## Description
Brief description of the changes.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Related Issues
Fixes #(issue number)

## How Has This Been Tested?
Describe the tests you ran to verify your changes.

## Checklist
- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
```

## Development Setup

### Prerequisites

- **Go 1.21+** installed
- **Docker** (optional, for running Loki locally)
- **Git** for version control

### Local Development

1. **Start Loki locally** (optional):
   ```bash
   docker run -d --name=loki -p 3100:3100 grafana/loki
   ```

2. **Run tests**:
   ```bash
   go test ./...
   ```

4. **Run linter**:
   ```bash
   golangci-lint run
   ```

5. **Clean build artifacts**:
   ```bash
   go clean -cache -testcache
   ```

## Style Guidelines

### Go Code Style
#### General Rules

- **Use `gofmt`** to format your code
- **Run `golangci-lint`** before committing
- **Write clear, self-documenting code**
- **Keep functions small and focused**
- **Use meaningful variable names**

#### Naming Conventions

```go
// ✅ Good
func NewLogger(config Config) (*Logger, error)
type HTTPClient struct{}
const MaxRetries = 3

// ❌ Bad
func new_logger(config Config) (*Logger, error)
type HttpClient struct{}
const max_retries = 3
```

#### Comments

- **Package comments**: Every package should have a package comment
- **Exported items**: All exported functions, types, and constants must be documented
- **Write in English** and follow Go documentation conventions
- **Start with the name** of the thing being described

```go
// Logger provides structured logging to Grafana Loki.
// It supports batching, multiple transports, and distributed tracing.
type Logger struct {
    // ...
}

// New creates a new Logger instance with the provided configuration.
// Returns an error if the configuration is invalid.
func New(config Config, opts ...Option) (*Logger, error) {
    // ...
}
```

#### Error Handling

- **Always check errors**
- **Wrap errors** with context using `fmt.Errorf`
- **Use custom error types** for specific error conditions

```go
// ✅ Good
func (l *Logger) log(level Level, msg string) error {
    if err := l.validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    // ...
}

// ❌ Bad
func (l *Logger) log(level Level, msg string) {
    l.validate() // Error ignored
    // ...
}
```

#### Concurrency

- **Use mutexes** to protect shared state
- **Avoid global mutable state**
- **Document goroutine lifecycle**

```go
// ✅ Good
func (l *Logger) Write(entry *Entry) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    // ...
}
```

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes

#### Examples

```
feat(transport): add retry mechanism with exponential backoff

Implements automatic retry for failed Loki requests with
configurable max retries and exponential backoff strategy.

Closes #42
```

```
fix(middleware): prevent trace_id from leaking between loggers

The WithFields method was doing shallow copy of config,
causing trace_id to be shared between logger instances.
Now performs deep copy of Labels map.

Fixes #58
```

```
docs(readme): add authentication examples

Added section showing how to use Loki with basic auth,
including environment variable best practices.
```

### Documentation

- **Update README.md** for user-facing changes
- **Update inline documentation** for code changes
- **Add examples** for new features
- **Keep TRACING.md and LABELS.md** up to date

## Review Process

1. **Automated checks** must pass (tests, linting)
2. **At least one maintainer** must review and approve
3. **Address all review comments** or explain why not
4. **Squash commits** if requested
5. **Merge** after approval

## Release Process

Releases are handled by maintainers:

1. **Version bump** following [Semantic Versioning](https://semver.org/)
2. **Create GitHub release** with release notes
3. **Tag release** in git

## Questions?

If you have questions:

- **Check existing documentation** (README, TRACING.md, LABELS.md)
- **Search existing issues** on GitHub
- **Open a new issue** with your question
- Be patient and respectful with maintainers

## Recognition

Contributors will be recognized in:
- GitHub contributors list
- Release notes for significant contributions
- README.md acknowledgments section (coming soon)

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT License).

---

Thank you for contributing to Loki Logger Go! 🎉
