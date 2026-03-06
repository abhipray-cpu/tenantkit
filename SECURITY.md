# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 1.0.x   | ✅ Current          |
| < 1.0   | ❌ Not supported    |

## Reporting a Vulnerability

If you discover a security vulnerability in TenantKit, please report it responsibly.

### How to Report

1. **Do NOT open a public GitHub issue** for security vulnerabilities.
2. Email **dumkaabhipray@gmail.com** with:
   - A description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggested fix (optional)

### What to Expect

- **Acknowledgement** within 48 hours
- **Assessment** within 1 week
- **Fix timeline** communicated after assessment
- **Credit** in release notes (unless you prefer anonymity)

### Scope

Security issues we care about:

- **Tenant isolation bypass** — Queries that leak data across tenants
- **SQL injection** via the query rewriter
- **Denial of service** via crafted queries that cause unbounded resource usage
- **Rate limiting / quota bypass** — Circumventing per-tenant limits
- **Context manipulation** — Forging or tampering with tenant context

### Out of Scope

- Issues in third-party dependencies (report upstream)
- Issues requiring physical access to the server
- Social engineering

## Security Best Practices

When using TenantKit:

1. **Always validate tenant IDs** before setting them in context
2. **Use TLS** for Redis connections (rate limiting, quota management)
3. **Audit bypass usage** — `WithoutTenantFiltering` should be used sparingly
4. **Enable PostgreSQL RLS** as a defense-in-depth layer alongside TenantKit
5. **Monitor metrics** for unusual cross-tenant query patterns
