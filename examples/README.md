# TenantKit Examples

Runnable examples demonstrating TenantKit features.

## basic/

Minimal example: wrap a SQLite database, insert data for two tenants, and query with tenant isolation.

```bash
cd basic && go run main.go
```

## http-middleware/

HTTP server with tenant middleware that extracts tenant ID from the `X-Tenant-ID` header.

```bash
cd http-middleware && go run main.go

# In another terminal:
curl -H "X-Tenant-ID: acme-corp" http://localhost:8080/users
```
