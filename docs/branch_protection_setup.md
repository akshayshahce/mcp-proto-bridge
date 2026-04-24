# Branch Protection Setup

This runbook applies enterprise branch protection using GitHub CLI.

## Prerequisites

1. `gh` CLI installed.
2. Authenticated `gh` session with repository admin permission.
3. Repository branch to protect (default is `main`).

## One-command setup

From repository root:

```sh
chmod +x scripts/apply-branch-protection.sh
./scripts/apply-branch-protection.sh
```

This applies required checks and merge safety controls.

Default required status checks:

1. `test`
2. `lint`
3. `enterprise-readiness`
4. `dependency-review`

## Optional overrides

```sh
REPO=owner/name BRANCH=main REVIEW_COUNT=2 REQUIRED_CHECKS="test,lint,enterprise-readiness,dependency-review" ./scripts/apply-branch-protection.sh
```

Supported environment variables:

1. `REPO` (format: `owner/name`)
2. `BRANCH` (default: `main`)
3. `REQUIRED_CHECKS` (comma-separated)
4. `REVIEW_COUNT` (default: `1`)
5. `ENFORCE_ADMINS` (default: `true`)
6. `STRICT_STATUS_CHECKS` (default: `true`)

## Verify applied configuration

```sh
gh api /repos/owner/name/branches/main/protection
```

## Troubleshooting

1. `Resource not accessible by integration`: token/user lacks admin rights on repository.
2. `Not Found`: repository name or branch is incorrect.
3. `Validation Failed`: one or more required status check names do not match actual GitHub check names.
