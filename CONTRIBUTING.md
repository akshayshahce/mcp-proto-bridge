# Contributing

Thank you for contributing to this project.

## Development setup

1. Install Go 1.24+.
2. Clone the repository.
3. Download dependencies:

```sh
go mod download
```

## Build, test, and lint

Run these before opening a pull request:

```sh
go test ./...
golangci-lint run
```

Optional integration test:

```sh
make integration-test
```

## Scope expectations for changes

1. Keep changes minimal and targeted.
2. Avoid unrelated refactors or style-only churn.
3. Do not change business behavior unless required for correctness.
4. Add or update tests when behavior changes.

## Pull request process

1. Open a pull request with a clear problem statement.
2. Include root cause, exact fix, and validation commands.
3. Ensure CI is green before requesting review.
4. Address review feedback with focused follow-up commits.

## Commit message guidance

Use concise, action-oriented commit titles. Example:

- fix: handle malformed MCP text payload fallback

## Reporting issues

Use issue templates for bug reports and feature requests. Include reproducible steps and expected behavior.

For security issues, follow the process in SECURITY.md.
