package main

import (
	"fmt"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"strings"
)

type SqliteColumn struct {
	Name string
	Type string
}
type SqliteColumns []SqliteColumn

func (s SqliteColumns) DeclarationString() string {
	var out []string
	for _, c := range s {
		out = append(out, fmt.Sprintf("%s %s", c.Name, c.Type))
	}

	return strings.Join(out, ", ")
}

func parsePluginSchema(ts *proto.TableSchema) (SqliteColumns, error) {
	cols := ts.Columns
	var out SqliteColumns

	for _, col := range cols {
		out = append(out, SqliteColumn{Name: col.Name, Type: GetMappedType(col.Type)})
	}
	return out, nil
}

func GetMappedType(in proto.ColumnType) string {
	switch in {
	case proto.ColumnType_BOOL, proto.ColumnType_INT:
		return "INT"
	case proto.ColumnType_DOUBLE:
		return "FLOAT"
	default:
		return "TEXT"
	}
}
