# Production Readiness Checklist

This checklist defines release gates for enterprise rollout of `mcp-proto-bridge`.

Status legend:

- PASS: gate is implemented and enforced by repository automation or committed policy.
- CONDITIONAL: gate is implemented but requires organization-level configuration.

## 1. Build and test quality gates

- PASS: unit tests run on every push/PR.
  - Evidence: `.github/workflows/ci.yml`
- PASS: lint runs on every push/PR.
  - Evidence: `.github/workflows/ci.yml`
- PASS: fuzz smoke runs in CI.
  - Evidence: `.github/workflows/enterprise-readiness.yml`
- PASS: benchmark smoke runs in CI for regressions visibility.
  - Evidence: `.github/workflows/enterprise-readiness.yml`

Pass criteria:

1. All required checks are green on target branch.
2. No skipped mandatory jobs for the merge commit.

## 2. Security and dependency gates

- PASS: Go vulnerability scanning enabled (`govulncheck`) in CI.
  - Evidence: `.github/workflows/enterprise-readiness.yml`
- PASS: PR dependency review enabled.
  - Evidence: `.github/workflows/dependency-review.yml`
- PASS: automated dependency update policy configured.
  - Evidence: `.github/dependabot.yml`
- PASS: security reporting policy documented.
  - Evidence: `SECURITY.md`

Pass criteria:

1. `govulncheck` finds no actionable vulnerabilities in shipped packages.
2. Dependency review reports no blocked severity additions on PRs.

## 3. Operational and reliability gates

- PASS: operational guidance documented.
  - Evidence: `docs/operational_readiness.md`
- PASS: behavior contract matrix documented.
  - Evidence: `docs/api_behavior_matrix.md`
- PASS: replay, counters, drift, and safety patterns are documented and tested.
  - Evidence: `docs/operational_readiness.md` + tests in `tests/`

Pass criteria:

1. Runbook links are present in service integration docs.
2. On-call can map top failure categories to deterministic remediation steps.

## 4. Change governance gates

- PASS: pull request template requires root-cause and validation evidence.
  - Evidence: `.github/pull_request_template.md`
- PASS: Copilot implementation constraints are documented for automated fix flows.
  - Evidence: `.github/copilot-instructions.md`
- CONDITIONAL: branch protection rules enforce required status checks and review policy.
  - Evidence: repository settings (outside source tree) + `docs/branch_protection_setup.md`

Pass criteria:

1. Every PR includes root cause, exact fix, and validation commands.
2. Protected branch settings require review and required checks before merge.

## 5. Enterprise rollout decision

Greenlight requires all PASS gates satisfied and all CONDITIONAL gates configured at org/repo settings.

Recommended required checks list:

1. `test`
2. `lint`
3. `enterprise-readiness`
4. `dependency-review`
