module github.com/abhipray-cpu/tenantkit/examples/http-middleware

go 1.24.0

require (
	github.com/abhipray-cpu/tenantkit/tenantkit v0.0.0
	github.com/mattn/go-sqlite3 v1.14.33
)

require (
	github.com/abhipray-cpu/tenantkit/domain v1.0.0 // indirect
	github.com/jackc/pgx/v5 v5.8.0 // indirect
)

replace github.com/abhipray-cpu/tenantkit/tenantkit => ../../tenantkit
