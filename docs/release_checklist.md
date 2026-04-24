# Release Checklist

Use this checklist before publishing an open source release.

## 1. Code health

1. `go test ./...` passes.
2. `golangci-lint run` passes.
3. Enterprise readiness workflow is green.

## 2. Security and dependencies

1. `govulncheck ./...` has no unresolved actionable vulnerabilities.
2. Dependency review is green for release PR.
3. No known high-severity dependency alerts remain open.

## 3. Documentation

1. README reflects current behavior and limitations.
2. Operational and production readiness docs are current.
3. Changelog or release notes include breaking and behavioral changes.

## 4. Compatibility and rollout

1. Public API changes are reviewed for backward compatibility.
2. Any migration guidance is documented.
3. Integration examples run successfully.

## 5. Publish and verify

1. Tag release from a green commit.
2. Publish release notes with root-cause context for major fixes.
3. Validate consumers can `go get` the tagged version.
