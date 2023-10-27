package main

import (
	"fmt"
	"strings"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
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

func mapSqliteOpToPluginOp(op sqlite.ConstraintOp) string {
	switch op {
	case sqlite.INDEX_CONSTRAINT_EQ:
		return "="
	case sqlite.INDEX_CONSTRAINT_GT:
		return ">"
	case sqlite.INDEX_CONSTRAINT_LE:
		return "<="
	case sqlite.INDEX_CONSTRAINT_LT:
		return "<"
	case sqlite.INDEX_CONSTRAINT_GE:
		return ">="
	}
	return "NOOP"
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
