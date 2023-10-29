package main

import (
	"fmt"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
)

func buildExecuteRequest(alias, table string, columns []string, quals map[string]*proto.Quals) *proto.ExecuteRequest {
	// we don't get any limit - hard code this to -1
	// needs investigation
	limit := int64(-1)

	fmt.Println("Quals:", quals)

	qc := proto.NewQueryContext(columns, quals, limit)
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
