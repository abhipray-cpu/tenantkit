module github.com/abhipray-cpu/tenantkit/testing/integration

go 1.21

replace github.com/abhipray-cpu/tenantkit/tenantkit => ../../tenantkit

replace github.com/abhipray-cpu/tenantkit/adapters/limiter-memory => ../../adapters/limiter-memory

replace github.com/abhipray-cpu/tenantkit/adapters/limiter-redis => ../../adapters/limiter-redis

replace github.com/abhipray-cpu/tenantkit/domain => ../../domain

require (
	github.com/abhipray-cpu/tenantkit/adapters/limiter-memory v0.0.0-00010101000000-000000000000
	github.com/abhipray-cpu/tenantkit/adapters/limiter-redis v0.0.0-00010101000000-000000000000
	github.com/abhipray-cpu/tenantkit/tenantkit v1.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.17.3 // indirect
)
