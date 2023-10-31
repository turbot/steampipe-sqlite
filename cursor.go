package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"go.riyazali.net/sqlite"
)

type PluginCursor struct {
	cursorCancel context.CancelFunc
	currentRow   int64
	stream       *plugin.LocalPluginStream
	currentItem  map[string]*proto.Column
	table        *PluginTable
}

// Filter is called by SQLite to restrict the number of rows returned by the virtual table.
// The implementation of this method should store the filter expression in the cursor object
// and then call Next() to advance the cursor to the first row that matches the filter.
func (p *PluginCursor) Filter(indexNumber int, indexString string, values ...sqlite.Value) error {
	fmt.Println("cursor.Filter:", indexNumber, indexString, values)
	defer fmt.Println("end cursor.Filter:", indexNumber, indexString, values)

	queryCtx, err := p.buildQueryContext(indexNumber, indexString, values...)
	if err != nil {
		return err
	}

	qualMap := p.buildQualMap(queryCtx, values...)

	execRequest := buildExecuteRequest(pluginAlias, p.table.name, queryCtx, qualMap)

	if err := pluginServer.CallExecute(execRequest, p.stream); err != nil {
		return err
	}

	p.currentRow = 0
	return nil
}

// Next is called by SQLite to advance the cursor to the next row in the result set.
// If an error occurs while advancing the cursor, this method should return an appropriate
// error code.
func (p *PluginCursor) Next() error {
	fmt.Println("cursor.Next")
	defer fmt.Println("end cursor.Next")
	item, err := p.stream.Recv()
	if err != nil {
		return err
	}
	if item == nil {
		p.currentRow = -1
		return sqlite.SQLITE_OK
	}

	p.currentItem = item.Row.Columns
	p.currentRow++

	return sqlite.SQLITE_OK
}

// Rowid is called by SQLite to retrieve the rowid for the current row.
func (p *PluginCursor) Rowid() (int64, error) {
	fmt.Println("cursor.RowId")
	defer fmt.Println("end cursor.RowId")
	return p.currentRow, nil
}

// Column is called by SQLite to retrieve the value for a column in the current row.
// The implementation of this method should call one of the ResultXXX() methods on the context
// to store the value for the column.
func (p *PluginCursor) Column(context *sqlite.VirtualTableContext, columnIdx int) error {
	column := p.table.tableSchema.Columns[columnIdx]

	switch column.Type {
	case proto.ColumnType_BOOL:
		if p.currentItem[column.Name].GetBoolValue() {
			context.ResultInt(1)
		} else {
			context.ResultInt(0)
		}
	case proto.ColumnType_INT:
		context.ResultInt(int(p.currentItem[column.Name].GetIntValue()))
	case proto.ColumnType_DOUBLE:
		context.ResultFloat(p.currentItem[column.Name].GetDoubleValue())
	case proto.ColumnType_STRING:
		context.ResultText(p.currentItem[column.Name].GetStringValue())
	case proto.ColumnType_JSON:
		context.ResultText(string(p.currentItem[column.Name].GetJsonValue()))
		context.ResultSubType(74) // 74 is JSON as per https://github.com/riyaz-ali/sqlite/blob/master/docs/RECIPES.md#json
	case proto.ColumnType_DATETIME, proto.ColumnType_TIMESTAMP:
		sqliteTimestampFormat := "2006-01-02 15:04:05.999"
		context.ResultText(p.currentItem[column.Name].GetTimestampValue().AsTime().Format(sqliteTimestampFormat))
	case proto.ColumnType_IPADDR:
		context.ResultText(p.currentItem[column.Name].GetIpAddrValue())
	case proto.ColumnType_CIDR:
		context.ResultText(p.currentItem[column.Name].GetCidrRangeValue())
	case proto.ColumnType_INET:
		context.ResultText(p.currentItem[column.Name].GetCidrRangeValue())
	case proto.ColumnType_LTREE:
		context.ResultText(p.currentItem[column.Name].GetLtreeValue())
	}

	return nil
}

// Eof is called by SQLite to determine if the cursor has reached the end of the result set.
func (p *PluginCursor) Eof() bool {
	fmt.Println("cursor.Eof")
	defer fmt.Println("end cursor.Eof")
	return p.currentRow < 0
}

// Close is called by SQLite to close the cursor.
// This method should release any resources held by the cursor.
func (p *PluginCursor) Close() error {
	fmt.Println("cursor.Close")
	defer fmt.Println("end cursor.Close")
	p.cursorCancel()
	return nil
}

func (p *PluginCursor) buildQueryContext(_ int, idxStr string, values ...sqlite.Value) (*QueryContext, error) {
	qc := new(QueryContext)
	if err := json.Unmarshal([]byte(idxStr), qc); err != nil {
		return nil, err
	}

	if qc.Limit != nil {
		// get the value at the given index
		v := values[qc.Limit.Idx]
		if v.Type() == sqlite.SQLITE_INTEGER {
			qc.Limit.Rows = v.Int64()
		} else {
			// this should never happen, but for some reason, the value is not an integer
			// so we will just ignore the limit
			qc.Limit = nil
		}
	}

	return qc, nil
}

func (p *PluginCursor) buildQualMap(qc *QueryContext, values ...sqlite.Value) map[string]*proto.Quals {
	// build the qual map
	qualMap := make(map[string]*proto.Quals)
	for i, qual := range qc.Quals {
		qualMap[qual.FieldName] = &proto.Quals{
			Quals: []*proto.Qual{
				{
					FieldName: qual.FieldName,
					Operator:  &proto.Qual_StringValue{StringValue: qual.Operator},
					Value:     getMappedQual(values[i]),
				},
			},
		}
	}
	return qualMap
}
