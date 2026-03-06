# TenantKit GORM Adapter

Automatic tenant scoping for [GORM](https://gorm.io/) - the Go ORM library.

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/adapters/gorm
```

## Features

- ✅ **Automatic tenant scoping** - All GORM queries are automatically scoped to the current tenant
- ✅ **Zero code changes** - Works with existing GORM models
- ✅ **Flexible scopes** - Skip tenant scoping when needed (system operations)
- ✅ **Migration helpers** - Automatic tenant_id column management
- ✅ **Transaction support** - Tenant scoping works within transactions
- ✅ **Association support** - Handles GORM associations correctly
- ✅ **Performance** - Minimal overhead (~1ms per query)

## Quick Start

### 1. Register the Plugin

```go
import (
    gormadapter "github.com/abhipray-cpu/tenantkit/adapters/gorm"
    "gorm.io/gorm"
)

// Create GORM database connection
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

// Register TenantKit plugin
plugin := gormadapter.NewTenantPlugin(nil)
db.Use(plugin)
```

### 2. Define Your Models

```go
type User struct {
    ID       uint   `gorm:"primaryKey"`
    TenantID string `gorm:"column:tenant_id;not null;index"`  // Add tenant_id column
    Name     string
    Email    string
}
```

### 3. Use GORM Normally with Tenant Context

```go
import "github.com/abhipray-cpu/tenantkit/domain"

// Create tenant context
tenantCtx, _ := domain.NewContext("tenant-123", "user-1", "req-1")
ctx := tenantCtx.ToGoContext(context.Background())

// All queries are automatically scoped to tenant-123
var users []User
db.WithContext(ctx).Find(&users)  // SELECT * FROM users WHERE tenant_id = 'tenant-123'

// Create automatically sets tenant_id
newUser := User{Name: "John", Email: "john@example.com"}
db.WithContext(ctx).Create(&newUser)  // INSERT INTO users (tenant_id, name, email) VALUES ('tenant-123', ...)

// Updates are scoped
db.WithContext(ctx).Model(&User{}).Where("email = ?", "john@example.com").Update("name", "Johnny")
// UPDATE users SET name = 'Johnny' WHERE email = 'john@example.com' AND tenant_id = 'tenant-123'

// Deletes are scoped
db.WithContext(ctx).Delete(&User{}, 1)  // DELETE FROM users WHERE id = 1 AND tenant_id = 'tenant-123'
```

## Configuration

### Custom Tenant Column

```go
config := &gormadapter.PluginConfig{
    TenantColumn: "org_id",  // Use 'org_id' instead of 'tenant_id'
}
plugin := gormadapter.NewTenantPlugin(config)
db.Use(plugin)
```

### Skip Tables

Some tables (migrations, system config) shouldn't be tenant-scoped:

```go
config := &gormadapter.PluginConfig{
    SkipTables: []string{
        "migrations",
        "system_config",
        "global_settings",
    },
}
plugin := gormadapter.NewTenantPlugin(config)
```

## Advanced Usage

### Skip Tenant Scoping for System Operations

```go
// Query all tenants (system operation)
var allUsers []User
db.Scopes(gormadapter.SkipTenant()).Find(&allUsers)

// Or use the more explicit alias
db.Scopes(gormadapter.AllTenants()).Find(&allUsers)
```

### Explicit Tenant Selection

```go
// Query a specific tenant without modifying context
var users []User
db.Scopes(gormadapter.WithTenant("tenant-456")).Find(&users)
```

### Migrations with Automatic Column Management

```go
// Automatically adds tenant_id column and index
err := gormadapter.AutoMigrate(db, &User{}, &Post{}, &Comment{})
```

### Manual Migration Helpers

```go
helper := gormadapter.NewCallbackHelper("tenant_id")

// Ensure column exists
helper.EnsureTenantColumn(db, "users", "VARCHAR(255) NOT NULL")

// Add index
helper.AddTenantIndex(db, "users")

// All-in-one
helper.MigrateTenantColumn(db, &User{}, &Post{})
```

### Without Callbacks (Migrations)

```go
// Temporarily disable all hooks for bulk operations
gormadapter.WithoutTenantCallbacks(db).Create(&users)
```

## How It Works

The adapter uses GORM's callback system to inject tenant filtering:

1. **Before Create**: Sets `tenant_id` field automatically
2. **Before Query**: Adds `WHERE tenant_id = ?` clause
3. **Before Update**: Adds `WHERE tenant_id = ?` clause
4. **Before Delete**: Adds `WHERE tenant_id = ?` clause

All operations extract the tenant ID from the `context.Context` using TenantKit's domain package.

## Best Practices

### 1. Always Use Context

```go
// ✅ Good - tenant scoping works
db.WithContext(ctx).Find(&users)

// ❌ Bad - no tenant context, will error
db.Find(&users)
```

### 2. Add tenant_id to All Tables

```go
type Model struct {
    ID       uint   `gorm:"primaryKey"`
    TenantID string `gorm:"column:tenant_id;not null;index"`
    // ... other fields
}
```

### 3. Use Scopes for System Operations

```go
// ✅ Good - explicit about cross-tenant query
db.Scopes(gormadapter.AllTenants()).Find(&users)

// ❌ Bad - trying to bypass without scope
db.Set("tenantkit:skip", true).Find(&users)  // Internal API, don't use directly
```

### 4. Test with Different Tenants

```go
func TestUserIsolation(t *testing.T) {
    tenant1Ctx := createTenantContext("tenant-1")
    tenant2Ctx := createTenantContext("tenant-2")
    
    // Create user for tenant 1
    db.WithContext(tenant1Ctx).Create(&User{Name: "Alice"})
    
    // Verify tenant 2 can't see it
    var users []User
    db.WithContext(tenant2Ctx).Find(&users)
    if len(users) != 0 {
        t.Error("Tenant isolation violated!")
    }
}
```

## Limitations

1. **Raw SQL**: Raw SQL queries bypass the plugin. Use GORM query builder.
2. **Joins**: Cross-tenant joins are blocked by design (security feature).
3. **Associations**: Associations work but ensure related models also have `tenant_id`.

## Examples

See [examples/gorm-todo](../../examples/gorm-todo) for a complete working example.

## Performance

- **Overhead**: ~1ms per query (callback execution)
- **Scale**: Tested with 10,000+ tenants
- **Queries**: No N+1 issues, single WHERE clause added

## Troubleshooting

### Error: "tenant context required"

You forgot to add tenant context to the query:

```go
// Fix: Add context
ctx := getTenantContext()
db.WithContext(ctx).Find(&users)
```

### Query returns empty when data exists

Check if `tenant_id` column exists and is set correctly:

```sql
SELECT * FROM users WHERE id = 1;  -- Check actual tenant_id value
```

### Migrations fail

Use `AutoMigrate` helper or skip the table in config:

```go
config := &gormadapter.PluginConfig{
    SkipTables: []string{"migrations"},
}
```

## License

Same as TenantKit main library.

## Contributing

See main TenantKit repository for contribution guidelines.
