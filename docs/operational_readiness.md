# Operational Readiness Guide

This guide focuses on using mcp-proto-bridge safely in production services.

## 1. Baseline service wiring

Enable these capabilities in service decode calls:

- hooks (`WithHooks`) for stage and drift events
- runtime counters (`WithRuntimeCounters`) for low-cardinality metrics
- safety limits (`WithSafetyLimits`) for payload boundaries
- policy and version rules (`WithDecodePolicy`, `WithVersionRules`) when needed

Use one shared option bundle per tool family to avoid configuration drift.

## 2. Metrics that should exist

From runtime counters, at minimum track:

- `decode.calls`, `decode.success`, `decode.failure`
- `decode_proto.calls`, `decode_proto.success`, `decode_proto.failure`
- `decode.auto_repair.applied`, `decode.auto_repair.passes`
- `drift.unknown_version`, `drift.ignored_tool_error`, `drift.ignored_no_payload`

Recommended derived signals:

- decode success rate
- decode failure rate by sentinel class
- drift event rate by tool/profile
- auto-repair usage rate

## 3. Logging and event quality

When handling `observe.Event`, ensure logs include:

- `kind`, `stage`, `target`, `profile`
- `provenance.resolved_version`
- `provenance.extractor_mode`
- `provenance.auto_repair_passes`
- `drift.type` for drift events

For decode failures, log both:

- wrapped `DecodeError` classification (`Stage`, `Category`, `Recoverability`)
- root cause via `errors.Is` / `errors.As`

## 4. Replay artifact handling

Replay artifacts are useful for deterministic reproduction of field issues.

Operational guidance:

- capture artifacts only on relevant failure classes
- redact sensitive fields before durable storage
- set retention windows by environment
- use artifacts in CI regression suites for high-value failure cases

## 5. Safety limits rollout

Rollout pattern:

1. start with observability only (no tight limits)
2. measure payload distributions in production-like traffic
3. set conservative limits above p95 or p99 shape sizes
4. tighten incrementally and monitor `ErrPayloadSafetyViolation`

## 6. Failure triage playbook

Map errors to actions:

- `tool_error`: inspect MCP provider response and permissions
- `no_payload`: inspect extractor mode and upstream response shape
- `invalid_json`: inspect text payload serialization
- `mapping`: inspect aliases and strict-mode unknown fields
- `validation`: inspect required fields and contract drift
- `safety`: inspect payload size/depth and limit configuration

## 7. Rollout gates for broad adoption

Before org-wide rollout, verify:

- integration matrix coverage in CI for success and failure scenarios
- stable behavior under strict and lenient modes
- deterministic replay artifacts for top failure classes
- dashboard and alerting in place for decode and drift metrics
- service runbook references this guide and the API behavior matrix

## 8. Fuzz and stress checks

Run fuzz targets periodically in CI/nightly jobs:

```sh
go test ./tests -run '^$' -fuzz FuzzDecodeNoPanic -fuzztime=10s
go test ./tests -run '^$' -fuzz FuzzDecodeCustomExtractorNoPanic -fuzztime=10s
```

Run decode throughput and parallel stress benchmarks:

```sh
go test ./tests -run '^$' -bench BenchmarkDecode -benchmem
```

Related references:

- [docs/api_behavior_matrix.md](docs/api_behavior_matrix.md)
- [integration/python_mcp/README.md](integration/python_mcp/README.md)