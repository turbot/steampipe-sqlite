package main

import (
	"fmt"
	"log"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

type Module struct {
	tableName   string
	columns     SQLiteColumns
	tableSchema *proto.TableSchema
	table       sqlite.VirtualTable
}

func NewModule(tableName string, columns SQLiteColumns, tableSchema *proto.TableSchema) *Module {
	return &Module{
		tableName:   tableName,
		columns:     columns,
		tableSchema: tableSchema,
		table:       &PluginTable{name: tableName, tableSchema: tableSchema},
	}
}

func (m *Module) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (sqlite.VirtualTable, error) {
	log.Println("[TRACE] Module.Connect start", m.tableName)
	defer log.Println("[TRACE] Module.Connect end", m.tableName)

	log.Println("[TRACE] Module.Connect table", m.tableName)
	return m.table, declare(fmt.Sprintf("CREATE TABLE %s(%s)", m.tableName, m.columns.DeclarationString()))
}
