module github.com/abhipray-cpu/tenantkit/adapters/metrics-noop

go 1.21

require github.com/abhipray-cpu/tenantkit/tenantkit v0.0.0

replace (
	github.com/abhipray-cpu/tenantkit/domain => ../../tenantkit/domain
	github.com/abhipray-cpu/tenantkit/tenantkit => ../../tenantkit
)
