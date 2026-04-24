#!/usr/bin/env bash
set -euo pipefail

# Applies enterprise branch protection with required CI checks.
# Prerequisites:
# - gh CLI installed and authenticated
# - caller has admin permission on the repository

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh CLI is required" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "error: gh is not authenticated; run: gh auth login" >&2
  exit 1
fi

REPO="${REPO:-}"
if [[ -z "$REPO" ]]; then
  REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)"
fi

if [[ -z "$REPO" ]]; then
  echo "error: could not resolve repository; set REPO=owner/name" >&2
  exit 1
fi

BRANCH="${BRANCH:-main}"
REQUIRED_CHECKS="${REQUIRED_CHECKS:-test,lint,enterprise-readiness,dependency-review}"
REVIEW_COUNT="${REVIEW_COUNT:-1}"
ENFORCE_ADMINS="${ENFORCE_ADMINS:-true}"
STRICT_STATUS_CHECKS="${STRICT_STATUS_CHECKS:-true}"

IFS=',' read -r -a checks <<<"$REQUIRED_CHECKS"
if [[ ${#checks[@]} -eq 0 ]]; then
  echo "error: REQUIRED_CHECKS must include at least one check context" >&2
  exit 1
fi

checks_json=""
for check in "${checks[@]}"; do
  trimmed="$(echo "$check" | sed 's/^ *//;s/ *$//')"
  if [[ -z "$trimmed" ]]; then
    continue
  fi
  if [[ -n "$checks_json" ]]; then
    checks_json+=" ,"
  fi
  checks_json+="\"$trimmed\""
done

if [[ -z "$checks_json" ]]; then
  echo "error: REQUIRED_CHECKS resolved to empty values" >&2
  exit 1
fi

payload="$(cat <<JSON
{
  \"required_status_checks\": {
    \"strict\": ${STRICT_STATUS_CHECKS},
    \"contexts\": [ ${checks_json} ]
  },
  \"enforce_admins\": ${ENFORCE_ADMINS},
  \"required_pull_request_reviews\": {
    \"dismiss_stale_reviews\": true,
    \"require_code_owner_reviews\": false,
    \"required_approving_review_count\": ${REVIEW_COUNT},
    \"require_last_push_approval\": true
  },
  \"restrictions\": null,
  \"required_linear_history\": true,
  \"allow_force_pushes\": false,
  \"allow_deletions\": false,
  \"required_conversation_resolution\": true
}
JSON
)"

echo "Applying branch protection to ${REPO} on branch ${BRANCH}"
echo "Required checks: ${REQUIRED_CHECKS}"

gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "/repos/${REPO}/branches/${BRANCH}/protection" \
  --input - <<<"$payload" >/dev/null

echo "Branch protection applied."
echo "Verify with: gh api /repos/${REPO}/branches/${BRANCH}/protection"
