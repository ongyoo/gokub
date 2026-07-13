---
name: gokub-verify-change
description: Review and verify changes in a GOKUB Go project before delivery. Use for code review, regression checks, security review, dependency changes, CI failures, release preparation, or when asked whether a feature, fix, refactor, or generated project is complete and safe to ship.
---

# Verify A GOKUB Change

Lead with concrete failures and risks. Do not treat successful compilation as complete
verification.

## Review

1. Read `.gokub.yaml`, `AGENTS.md`, and the change diff or touched files.
2. Trace behavior across handler, service, repository, adapter, and process lifecycle.
3. Check context propagation, error wrapping, cleanup, timeouts, concurrency, and
   graceful shutdown.
4. Check that secrets are absent from source, logs, fixtures, and committed env files.
5. Check request validation, authorization boundaries, safe query construction, and
   generic client-facing internal errors.
6. Confirm capability and recipe state matches generated files.

## Execute

Run the project-native equivalents of:

```bash
gofmt -l .
go test -race ./...
go vet ./...
go build ./...
gokub doctor
gokub score
gokub graph
gokub upgrade --check
```

For HTTP changes, exercise health endpoints and affected routes. For dependency
changes, run `go mod tidy`, inspect `go.mod` and `go.sum`, and ensure no unexplained
module churn. For container or CI changes, validate YAML and build definitions.
Address score recommendations caused by the change or explain why they are not
applicable.

## Report

Report findings in severity order with file and line references. Distinguish verified
behavior from inference. State skipped commands and their blockers. If no issue is
found, say so and identify remaining integration or environment risk.
