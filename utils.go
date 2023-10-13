package main

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
)

func BuildExecuteRequest(alias, table string, tableSchema *proto.TableSchema) *proto.ExecuteRequest {
	var quals map[string]*proto.Quals
	limit := int64(-1)

	qc := proto.NewQueryContext(tableSchema.GetColumnNames(), quals, limit)
	ecd := proto.ExecuteConnectionData{
		Limit:        qc.Limit,
		CacheEnabled: false,
	}
	req := proto.ExecuteRequest{
		Table:                 table,
		QueryContext:          qc,
		CallId:                grpc.BuildCallId(),
		Connection:            alias,
		TraceContext:          nil,
		ExecuteConnectionData: make(map[string]*proto.ExecuteConnectionData),
	}
	req.ExecuteConnectionData[alias] = &ecd

	return &req
}
