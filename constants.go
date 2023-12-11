package main

const (
	SQLITE_INDEX_CONSTRAINT_LIMIT = 73
	SQLITE_TIMESTAMP_FORMAT       = "2006-01-02 15:04:05.999"
	SQLITE_DATEONLY_FORMAT        = "2006-01-02"
	EnvCacheEnabled               = "STEAMPIPE_CACHE"
	EnvCacheMaxTTL                = "STEAMPIPE_CACHE_MAX_TTL"
)

type SchemaMode string

func (sm SchemaMode) Equals(s string) bool {
	return string(sm) == s
}

const (
	SCHEMA_MODE_STATIC  SchemaMode = "static"
	SCHEMA_MODE_DYNAMIC SchemaMode = "dynamic"
)
