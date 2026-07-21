# Contributing to CodeScope

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Run vet: `go vet ./...`
6. Submit a pull request

## Adding a New Language

1. Create a new parser in `internal/parser/`
2. Implement the `Parser` interface
3. Register the language in `internal/detector/detector.go`
4. Add language extensions to `scanner.go`
5. Add tests

## Code Style

- Follow standard Go conventions (`gofmt`)
- No external dependencies
- All analysis must be deterministic
- No AI, no cloud, no telemetry
