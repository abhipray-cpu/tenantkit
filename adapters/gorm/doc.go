package gormadapter

// Package gormadapter provides a GORM plugin for automatic tenant filtering.
//
// It registers GORM callbacks that inject tenant conditions into all
// queries, creating, updating, and deleting operations.
//
// # Usage
//
//	plugin := gormadapter.NewTenantPlugin(gormadapter.PluginConfig{
//	    TenantColumn: "tenant_id",
//	})
//	gormDB.Use(plugin)
