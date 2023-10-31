package main

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
)

func buildExecuteRequest(alias, table string, ctx *QueryContext, quals map[string]*proto.Quals) *proto.ExecuteRequest {
	limitRows := int64(-1)
	if ctx.Limit != nil {
		limitRows = ctx.Limit.Rows
	}
	qc := proto.NewQueryContext(ctx.Columns, quals, limitRows)
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
