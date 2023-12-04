package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/turbot/steampipe-plugin-sdk/v5/anywhere"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

// PluginCursor implements the sqlite/virtual_table.Cursor interface.
// It is used to allow the SQLite core to interact with the virtual table and retrieve rows.
type PluginCursor struct {
	cursorCancel context.CancelFunc
	currentRow   int64
	stream       *anywhere.LocalPluginStream
	currentItem  map[string]*proto.Column
	table        *PluginTable
}

// NewPluginCursor creates a new cursor for a plugin table.
func NewPluginCursor(ctx context.Context, table *PluginTable) *PluginCursor {
	_, cancel := context.WithCancel(ctx)
	return &PluginCursor{
		table:        table,
		stream:       anywhere.NewLocalPluginStream(ctx),
		cursorCancel: cancel,
		currentRow:   0,
		currentItem:  make(map[string]*proto.Column),
	}
}

// Filter is called by SQLite to restrict the number of rows returned by the virtual table.
// The implementation of this method should store the filter expression in the cursor object
// and then call Next() to advance the cursor to the first row that matches the filter.
func (p *PluginCursor) Filter(indexNumber int, indexString string, values ...sqlite.Value) error {
	log.Println("[DEBUG] cursor.Filter:", p.table.name, indexNumber, indexString, values)
	defer log.Println("[DEBUG] end cursor.Filter:", p.table.name, indexNumber, indexString, values)

	queryCtx, err := p.buildQueryContext(indexNumber, indexString, values...)
	if err != nil {
		return err
	}

	qualMap, err := p.buildQualMap(queryCtx, values...)
	if err != nil {
		return err
	}

	execRequest := buildExecuteRequest(pluginAlias, p.table.name, queryCtx, qualMap)

	pluginServer.CallExecuteAsync(execRequest, p.stream)

	p.currentRow = 0
	return p.Next()
}

// Next is called by SQLite to advance the cursor to the next row in the result set.
// If an error occurs while advancing the cursor, this method should return an appropriate
// error code.
func (p *PluginCursor) Next() error {
	log.Println("[DEBUG] cursor.Next")
	defer log.Println("[DEBUG] end cursor.Next")
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
	log.Println("[DEBUG] cursor.RowId")
	defer log.Println("[DEBUG] end cursor.RowId")
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
		context.ResultText(p.currentItem[column.Name].GetTimestampValue().AsTime().Format(SQLITE_TIMESTAMP_FORMAT))
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
	log.Println("[DEBUG] cursor.Eof")
	defer log.Println("[DEBUG] end cursor.Eof")
	return p.currentRow < 0
}

// Close is called by SQLite to close the cursor.
// This method should release any resources held by the cursor.
func (p *PluginCursor) Close() error {
	log.Println("[DEBUG] cursor.Close")
	defer log.Println("[DEBUG] end cursor.Close")
	p.cursorCancel()
	return nil
}

func (p *PluginCursor) buildQueryContext(_ int, idxStr string, values ...sqlite.Value) (*QueryContext, error) {
	log.Println("[DEBUG] cursor.buildQueryContext")
	defer log.Println("[DEBUG] end cursor.buildQueryContext")

	qc := new(QueryContext)
	if err := json.Unmarshal([]byte(idxStr), qc); err != nil {
		return nil, err
	}

	p.extractLimitForQueryContext(qc, values...)

	return qc, nil
}

func (p *PluginCursor) extractLimitForQueryContext(qc *QueryContext, values ...sqlite.Value) {
	log.Println("[DEBUG] cursor.extractLimitForQueryContext")
	defer log.Println("[DEBUG] end cursor.extractLimitForQueryContext")

	if qc.Limit != nil {
		// get the value at the given index
		v := values[qc.Limit.ArgvIdx-1]
		if v.Type() == sqlite.SQLITE_INTEGER {
			qc.Limit.Rows = v.Int64()
		} else {
			// this should never happen, but for some reason, the value is not an integer
			// so we will just ignore the limit
			qc.Limit = nil
		}
	}
}

func (p *PluginCursor) buildQualMap(qc *QueryContext, values ...sqlite.Value) (map[string]*proto.Quals, error) {
	log.Println("[DEBUG] cursor.buildQualMap")
	defer log.Println("[DEBUG] end cursor.buildQualMap")

	// build the qual map
	qualMap := make(map[string]*proto.Quals)
	for _, qual := range qc.Quals {
		mappedValue, err := getMappedQualValue(values[qual.ArgvIndex-1], qual)
		if err != nil {
			return nil, err
		}
		qualMap[qual.FieldName] = &proto.Quals{
			Quals: []*proto.Qual{
				{
					FieldName: qual.FieldName,
					Operator:  &proto.Qual_StringValue{StringValue: qual.Operator},
					Value:     mappedValue,
				},
			},
		}
	}
	return qualMap, nil
}
