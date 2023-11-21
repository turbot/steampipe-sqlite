package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"time"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SQLiteColumn struct {
	Name string
	Type string
}
type SQLiteColumns []SQLiteColumn

func (s SQLiteColumns) DeclarationString() string {
	var out []string
	for _, c := range s {
		out = append(out, fmt.Sprintf("%s %s", c.Name, c.Type))
	}

	return strings.Join(out, ", ")
}

// getPluginOperator converts a sqlite.ConstraintOp to a QualOperator
func getPluginOperator(op sqlite.ConstraintOp) *QualOperator {
	log.Println("[TRACE] getPluginOperator start", op)
	defer log.Println("[TRACE] getPluginOperator end", op)

	cost := &QualOperator{
		Op:   "NOOP",
		Cost: math.MaxFloat64,
	}
	switch op {
	case sqlite.INDEX_CONSTRAINT_EQ:
		cost.Op = "="
		cost.Cost = 1
	case sqlite.INDEX_CONSTRAINT_GT:
		cost.Op = ">"
		cost.Cost = 10
	case sqlite.INDEX_CONSTRAINT_GE:
		cost.Op = ">="
		cost.Cost = 10
	case sqlite.INDEX_CONSTRAINT_LE:
		cost.Op = "<="
		cost.Cost = 10
	case sqlite.INDEX_CONSTRAINT_LT:
		cost.Op = "<"
		cost.Cost = 10
		// we should extend this to include LIKE, GLOB, REGEXP, MATCH and others
	}
	return cost
}

// getSQLiteColumnsFromTableSchema converts a proto.TableSchema to a SQLiteColumns
// which can be used to create a SQLite table
func getSQLiteColumnsFromTableSchema(ts *proto.TableSchema) SQLiteColumns {
	cols := ts.Columns
	var out SQLiteColumns

	for _, col := range cols {
		out = append(out, SQLiteColumn{Name: col.Name, Type: getMappedType(col.Type)})
	}
	return out
}

// getMappedType converts a proto.ColumnType to a SQLite type
func getMappedType(in proto.ColumnType) string {
	switch in {
	case proto.ColumnType_BOOL, proto.ColumnType_INT:
		return "INT"
	case proto.ColumnType_DOUBLE:
		return "FLOAT"
	default:
		// everything else is a string as far as SQLite is concerned
		return "TEXT"
	}
}

// getMappedQualValue converts a sqlite.Value to a proto.QualValue
// based on the type of the column definition of the qual
func getMappedQualValue(v sqlite.Value, qual *Qual) (*proto.QualValue, error) {
	switch v.Type() {
	case sqlite.SQLITE_INTEGER:
		return getMappedIntValue(v.Int64(), qual)
	case sqlite.SQLITE_TEXT:
		return getMappedStringValue(v.Text(), qual)
	case sqlite.SQLITE_FLOAT:
		return &proto.QualValue{Value: &proto.QualValue_DoubleValue{DoubleValue: v.Float()}}, nil
	case sqlite.SQLITE_NULL:
		return &proto.QualValue{Value: nil}, nil
	default:
		// default to a string
		return &proto.QualValue{Value: &proto.QualValue_StringValue{StringValue: v.Text()}}, nil
	}
}

// getMappedStringValue converts a string to a proto.QualValue
// based on the type of the column definition of the qual
func getMappedStringValue(v string, q *Qual) (*proto.QualValue, error) {
	switch q.ColumnDefinition.GetType() {
	case proto.ColumnType_IPADDR, proto.ColumnType_INET:
		ip := net.ParseIP(v)
		if ip != nil {
			return &proto.QualValue{
				Value: &proto.QualValue_InetValue{
					InetValue: &proto.Inet{
						Addr: ip.String(),
					},
				},
			}, nil
		}
		return nil, fmt.Errorf("could not parse '%s' as IP ADDR", v)
	case proto.ColumnType_CIDR:
		_, _, err := net.ParseCIDR(v)
		if err == nil {
			return nil, err
		}
		return &proto.QualValue{
			Value: &proto.QualValue_InetValue{
				InetValue: &proto.Inet{
					Cidr: v,
				},
			},
		}, nil
	case proto.ColumnType_LTREE:
		return &proto.QualValue{Value: &proto.QualValue_LtreeValue{LtreeValue: v}}, nil
	case proto.ColumnType_JSON:
		return &proto.QualValue{Value: &proto.QualValue_JsonbValue{JsonbValue: v}}, nil
	case proto.ColumnType_DATETIME, proto.ColumnType_TIMESTAMP:
		timestamp, err := time.Parse(SQLITE_TIMESTAMP_FORMAT, v)
		if err != nil {
			return nil, err
		}
		return &proto.QualValue{
			Value: &proto.QualValue_TimestampValue{
				TimestampValue: timestamppb.New(timestamp),
			},
		}, nil
	}
	return &proto.QualValue{Value: &proto.QualValue_StringValue{StringValue: v}}, nil
}

// getMappedIntValue converts an int64 to a proto.QualValue
// based on the type of the column definition of the qual
func getMappedIntValue(v int64, q *Qual) (*proto.QualValue, error) {
	switch q.ColumnDefinition.GetType() {
	case proto.ColumnType_BOOL:
		return &proto.QualValue{Value: &proto.QualValue_BoolValue{BoolValue: v != 0}}, nil
	default:
		return &proto.QualValue{Value: &proto.QualValue_Int64Value{Int64Value: v}}, nil
	}
}
