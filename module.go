package main

import (
	"context"
	"fmt"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"go.riyazali.net/sqlite"
)

type Module struct {
	tableName   string
	Columns     SqliteColumns
	TableSchema *proto.TableSchema
}

func NewModule(tableName string, columns SqliteColumns, tableSchema *proto.TableSchema) *Module {
	return &Module{
		tableName:   tableName,
		Columns:     columns,
		TableSchema: tableSchema,
	}
}

func (m *Module) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (sqlite.VirtualTable, error) {
	table := &PluginTable{Name: m.tableName, tableSchema: m.TableSchema}
	return table, declare(fmt.Sprintf("CREATE TABLE %s(%s)", m.tableName, m.Columns.DeclarationString()))
}

type PluginTable struct {
	Name        string
	tableSchema *proto.TableSchema
}

func (p *PluginTable) BestIndex(_ *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	// TODO: Figure out how to improve this using Quals etc
	return &sqlite.IndexInfoOutput{EstimatedCost: 1000}, nil
}

func (p *PluginTable) Open() (sqlite.VirtualCursor, error) {
	ctx := context.Background()

	cursor := &PluginCursor{
		stream:      plugin.NewLocalPluginStream(ctx),
		currentRow:  0,
		currentItem: make(map[string]*proto.Column),
		tableSchema: p.tableSchema,
	}
	err := pluginServer.CallExecute(BuildExecuteRequest(pluginAlias, p.Name, p.tableSchema), cursor.stream)
	return cursor, err
}

func (p *PluginTable) Disconnect() error {
	return nil
}

func (p *PluginTable) Destroy() error {
	return nil
}

type PluginCursor struct {
	currentRow  int64
	stream      *plugin.LocalPluginStream
	currentItem map[string]*proto.Column
	tableSchema *proto.TableSchema
}

func (p *PluginCursor) Filter(i int, s string, value ...sqlite.Value) error {
	p.currentRow = 0
	return p.Next()
}

func (p *PluginCursor) Next() error {
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

func (p *PluginCursor) Rowid() (int64, error) {
	return p.currentRow, nil
}

func (p *PluginCursor) Column(context *sqlite.VirtualTableContext, i int) error {
	column := p.tableSchema.Columns[i]

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

func (p *PluginCursor) Eof() bool {
	return p.currentRow < 0
}

func (p *PluginCursor) Close() error {
	return nil
}
