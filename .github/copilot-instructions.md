# Copilot Working Instructions

When resolving CI failures in this repository, follow these rules:

1. Fix only what is required for failing lint/test checks.
2. Keep changes minimal, targeted, and production-safe.
3. Do not refactor unrelated code or perform broad style churn.
4. Do not change business behavior unless the failing check proves it is necessary.
5. Update tests only when tests are incorrect or stale, not to hide real defects.
6. Prefer deterministic fixes over retries or flaky workarounds.
7. Preserve existing public APIs unless failure explicitly requires change.
8. Verify with the same checks that failed in CI before opening a PR.
9. In the PR description, clearly explain:
   - root cause
   - exact fix
   - validation commands and outcomes

Additional guidance:

- Keep commit diffs small and readable.
- Add concise comments only where logic is non-obvious.
- Avoid introducing new dependencies unless absolutely necessary.