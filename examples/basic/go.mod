module github.com/abhipray-cpu/tenantkit/examples/basic

go 1.24.0

replace github.com/abhipray-cpu/tenantkit/tenantkit => ../../tenantkit

require (
	github.com/abhipray-cpu/tenantkit/tenantkit v0.0.0-00010101000000-000000000000
	github.com/mattn/go-sqlite3 v1.14.33
)

require (
	github.com/abhipray-cpu/tenantkit/domain v1.0.0 // indirect
	github.com/jackc/pgx/v5 v5.8.0 // indirect
)
