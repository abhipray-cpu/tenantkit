---
name: Bug Report
about: Create a report to help us improve
title: '[BUG] '
labels: bug
assignees: abhipray-cpu
---

## Description

A clear and concise description of what the bug is.

## To Reproduce

Steps to reproduce the behavior:

1. Setup '...'
2. Call '...'
3. With parameters '...'
4. See error

## Expected Behavior

A clear and concise description of what you expected to happen.

## Actual Behavior

What actually happened.

## Code Sample

```go
// Minimal code sample that reproduces the issue
wrappedDB := tenantkit.Wrap(db, "tenant_id", tenantkit.Config{})
ctx := tenantkit.WithTenant(context.Background(), "tenant-1")
// ...
```

## Environment

- **Go version**: [e.g., 1.21]
- **tenantkit version**: [e.g., v1.0.0]
- **Database**: [e.g., PostgreSQL 16]
- **OS**: [e.g., macOS 14, Ubuntu 22.04]
- **Architecture**: [e.g., amd64, arm64]

## Error Output

```
Paste any error messages, stack traces, or logs here
```

## Additional Context

Add any other context about the problem here.

## Possible Solution

If you have ideas on how to fix the bug, please describe them here.
