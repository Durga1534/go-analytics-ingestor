# Contributing to Go Analytics Ingestor

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to this project.

## Code of Conduct

Be respectful and professional. We're committed to providing a welcoming and inclusive environment for all contributors.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/go-analytics-ingestor.git
   cd go-analytics-ingestor
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/Durga1534/go-analytics-ingestor.git
   ```
4. **Create a feature branch**:
   ```bash
   git checkout -b feat/your-feature-name
   ```

## Development Setup

```bash
# Install dependencies
go mod download

# Set up environment
cp .env.example .env

# Run with Docker Compose (recommended)
docker-compose up

# Or run locally
make dev
```

## Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - A new feature
- `fix:` - A bug fix
- `docs:` - Documentation only changes
- `style:` - Changes that don't affect code meaning (formatting, semicolons, etc.)
- `refactor:` - A code change that neither fixes a bug nor adds a feature
- `perf:` - A code change that improves performance
- `test:` - Adding or updating tests
- `chore:` - Changes to build process or dependencies
- `ci:` - Changes to CI/CD configuration

### Examples
```bash
git commit -m "feat: add event retry logic"
git commit -m "fix: resolve race condition in worker"
git commit -m "docs: update API documentation"
```

## Code Style

We use standard Go formatting and conventions:

```bash
# Format your code
make fmt

# Run linter
make lint

# Run security scanner
go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
```

### Guidelines
- Use meaningful variable and function names
- Keep functions small and focused (single responsibility)
- Add comments for exported functions and complex logic
- Follow [Effective Go](https://golang.org/doc/effective_go)

## Testing

All changes must include tests:

```bash
# Run all tests
make test

# Run specific test
go test -v ./internal/handlers

# Run with coverage
go test -cover ./...
```

### Test Coverage
- Aim for >80% code coverage
- Test both success and failure cases
- Use table-driven tests for multiple scenarios

## Pull Request Process

1. **Keep PRs focused** - One feature or fix per PR
2. **Write clear descriptions** - Explain what and why
3. **Update documentation** - Keep README and comments current
4. **Add tests** - Include unit tests for new features
5. **Ensure CI passes** - All automated checks must pass
6. **Request review** - Tag maintainers for review

### PR Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added (where needed)
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests pass locally
- [ ] No new warnings generated

## Project Structure

```
.
├── cmd/server/           # Application entry point
├── internal/             # Private application code
│   ├── cache/           # Redis layer
│   ├── config/          # Configuration
│   ├── handlers/        # HTTP handlers
│   ├── logger/          # Logging
│   ├── models/          # Data types
│   ├── persistence/     # Database layer
│   └── worker/          # Business logic
├── scripts/             # Utility scripts
└── .github/             # GitHub configuration
    └── workflows/       # CI/CD pipelines
```

## Common Tasks

### Adding a New Feature

1. Create feature branch: `git checkout -b feat/my-feature`
2. Implement feature in appropriate package
3. Add tests: `internal/handlers/my_feature_test.go`
4. Update documentation
5. Run checks: `make fmt lint test`
6. Commit with conventional message
7. Push and create PR

### Adding a New Handler

```go
// internal/handlers/my_handler.go
package handlers

import "net/http"

func MyHandler(/* dependencies */) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation
    }
}
```

Then register in `cmd/server/main.go`:
```go
http.HandleFunc("/my-endpoint", handlers.MyHandler(...))
```

### Adding a Database Migration

1. Update `scripts/init.sql` with new schema
2. Add migration function if needed
3. Test with Docker Compose
4. Document in PR

## Reporting Issues

When reporting bugs, include:

- Minimal code example to reproduce
- Expected and actual behavior
- Go version (`go version`)
- Operating system
- Relevant logs/error messages

## Questions?

Open an issue with tag `[QUESTION]` or start a discussion.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing! 🙏
