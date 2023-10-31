package main

import (
	"fmt"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

type Module struct {
	tableName   string
	columns     SqliteColumns
	tableSchema *proto.TableSchema
}

func NewModule(tableName string, columns SqliteColumns, tableSchema *proto.TableSchema) *Module {
	return &Module{
		tableName:   tableName,
		columns:     columns,
		tableSchema: tableSchema,
	}
}

func (m *Module) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (sqlite.VirtualTable, error) {
	table := &PluginTable{name: m.tableName, tableSchema: m.tableSchema}
	return table, declare(fmt.Sprintf("CREATE TABLE %s(%s)", m.tableName, m.columns.DeclarationString()))
}
