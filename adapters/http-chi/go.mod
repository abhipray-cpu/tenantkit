module github.com/abhipray-cpu/tenantkit/adapters/http-chi

go 1.21

replace github.com/abhipray-cpu/tenantkit/domain => ../../tenantkit/domain

require (
	github.com/abhipray-cpu/tenantkit/domain v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.0.12
)
