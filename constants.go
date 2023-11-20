package main

const (
	SQLITE_INDEX_CONSTRAINT_LIMIT = 73
	SQLITE_TIMESTAMP_FORMAT       = "2006-01-02 15:04:05.999"
)

type SchemaMode string

func (sm SchemaMode) Equals(s string) bool {
	return string(sm) == s
}

const (
	SCHEMA_MODE_STATIC  SchemaMode = "static"
	SCHEMA_MODE_DYNAMIC SchemaMode = "dynamic"
)
