package main

import (
	"fmt"
	"math"
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

func mapSqliteOpToPluginOpAndCost(op sqlite.ConstraintOp) (string, float64) {
	switch op {
	case sqlite.INDEX_CONSTRAINT_EQ:
		return "=", 1
	case sqlite.INDEX_CONSTRAINT_GT:
		return ">", 10
	case sqlite.INDEX_CONSTRAINT_LE:
		return "<=", 10
	case sqlite.INDEX_CONSTRAINT_LT:
		return "<", 10
	case sqlite.INDEX_CONSTRAINT_GE:
		return ">=", 10
	}
	return "NOOP", math.MaxFloat64
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

func getMappedQual(v sqlite.Value) *proto.QualValue {
	switch v.Type() {
	case sqlite.SQLITE_INTEGER:
		return &proto.QualValue{Value: &proto.QualValue_Int64Value{Int64Value: v.Int64()}}
	case sqlite.SQLITE_FLOAT:
		return &proto.QualValue{Value: &proto.QualValue_DoubleValue{DoubleValue: v.Float()}}
	case sqlite.SQLITE_TEXT:
		return &proto.QualValue{Value: &proto.QualValue_StringValue{StringValue: v.Text()}}
	case sqlite.SQLITE_NULL:
		return &proto.QualValue{Value: nil}
	}
	return nil
}
