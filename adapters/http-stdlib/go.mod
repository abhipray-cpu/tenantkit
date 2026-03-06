module github.com/abhipray-cpu/tenantkit/adapters/http-stdlib

go 1.21

require (
	github.com/abhipray-cpu/tenantkit/tenantkit v0.0.0
	github.com/abhipray-cpu/tenantkit/domain v0.0.0
	github.com/abhipray-cpu/tenantkit/adapters/limiter-memory v0.0.0
)

replace github.com/abhipray-cpu/tenantkit/tenantkit v0.0.0 => ../../tenantkit

replace github.com/abhipray-cpu/tenantkit/domain v0.0.0 => ../../tenantkit/domain

replace github.com/abhipray-cpu/tenantkit/adapters/limiter-memory v0.0.0 => ../limiter-memory
