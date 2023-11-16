package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

// ConfigureFn implements a custom scalar sql function
// that allows the user to configure the plugin connection
type ConfigureFn struct {
	api *sqlite.ExtensionApi
}

func NewConfigureFn(api *sqlite.ExtensionApi) *ConfigureFn {
	return &ConfigureFn{
		api: api,
	}
}

func (m *ConfigureFn) Args() int           { return 1 }
func (m *ConfigureFn) Deterministic() bool { return true }
func (m *ConfigureFn) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	log.Println("[TRACE] ConfigureFn.Apply start")
	defer log.Println("[TRACE] ConfigureFn.Apply end")

	var config string
	var err error
	log.Println("[TRACE] getting config")
	if config, err = m.getConfig(values...); err != nil {
		ctx.ResultError(err)
		return
	}

	// Set Connection Config
	err = m.setConnectionConfig(config)
	if err != nil {
		ctx.ResultError(err)
		return
	}
}

// getConfig returns the config string from the first argument
func (m *ConfigureFn) getConfig(values ...sqlite.Value) (config string, err error) {
	log.Println("[TRACE] ConfigureFn.getConfig start")
	defer log.Println("[TRACE] ConfigureFn.getConfig end")

	if len(values) > 1 {
		return "", errors.New("expected a single argument")
	}

	switch {
	case values[0].Type() == sqlite.SQLITE_TEXT:
		config = values[0].Text()
	case values[0].Type() == sqlite.SQLITE_BLOB:
		config = string(values[0].Blob())
	default:
		return "", (errors.New("expected a TEXT or BLOB argument"))
	}
	return config, nil
}

// setConnectionConfig sets the connection config for the plugin
func (m *ConfigureFn) setConnectionConfig(config string) error {
	log.Println("[TRACE] ConfigureFn.setConnectionConfig start")
	defer log.Println("[TRACE] ConfigureFn.setConnectionConfig end")

	pluginName := fmt.Sprintf("steampipe-plugin-%s", pluginAlias)

	c := &proto.ConnectionConfig{
		Connection:      pluginAlias,
		Plugin:          pluginName,
		PluginShortName: pluginAlias,
		Config:          config,
		PluginInstance:  pluginName,
	}

	if currentSchema != nil {
		log.Println("[TRACE] ConfigureFn.setConnectionConfig: updating connection config")
		// send an update request to the plugin server
		cs := []*proto.ConnectionConfig{c}
		req := &proto.UpdateConnectionConfigsRequest{Changed: cs}
		_, err := pluginServer.UpdateConnectionConfigs(req)
		if err != nil {
			return err
		}
	} else {
		log.Println("[TRACE] ConfigureFn.setConnectionConfig: setting connection config")
		// set the config in the plugin server
		cs := []*proto.ConnectionConfig{c}
		req := &proto.SetAllConnectionConfigsRequest{
			Configs:        cs,
			MaxCacheSizeMb: 32,
		}
		_, err := pluginServer.SetAllConnectionConfigs(req)
		if err != nil {
			return err
		}
	}

	// fetch the schema
	// we cannot use the global currentSchema variable here
	// because it may not have been loaded yet at all
	schema, err := getSchema()
	if err != nil {
		return err
	}

	log.Println("[TRACE] ConfigureFn.setConnectionConfig: schema fetched successfully")

	// we should also trigger a schema refresh after this call for dynamic backends
	if SCHEMA_MODE_DYNAMIC.Equals(schema.Mode) {
		// drop the existing tables - if they have been created
		if err := m.dropCurrent(); err != nil {
			return err
		}

		// create the tables for the new dynamic schema
		if err := setupTables(schema, m.api); err != nil {
			return err
		}
		currentSchema = schema
	}

	return err
}

func (m *ConfigureFn) dropCurrent() error {
	if currentSchema != nil {
		sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
			conn := api.Connection()
			for tableName := range currentSchema.GetSchema() {
				log.Println("[TRACE] ConfigureFn.dropCurrent: dropping table", tableName)
				q := fmt.Sprintf("DROP TABLE %s", tableName)
				log.Println("[TRACE] ConfigureFn.dropCurrent: executing query", q)
				err := conn.Exec(q, nil)
				if err != nil {
					log.Println("[ERROR] ConfigureFn.dropCurrent: error dropping table", tableName, err)
					return sqlite.SQLITE_ERROR, err
				}
			}
			return sqlite.SQLITE_OK, nil
		})
	}
	return nil
}

// getSchema returns the schema for the plugin
func getSchema() (*proto.Schema, error) {
	log.Println("[TRACE] getSchema start")
	defer log.Println("[TRACE] getSchema end")

	// Get Plugin Schema
	sRequest := &proto.GetSchemaRequest{Connection: pluginAlias}
	s, err := pluginServer.GetSchema(sRequest)
	if err != nil {
		return nil, err
	}
	return s.GetSchema(), nil
}

// setupTables sets up the schema tables for the plugin
// it fetched the schema from the plugin and then maps it to SQLite tables
func setupTables(schema *proto.Schema, api *sqlite.ExtensionApi) error {
	log.Println("[TRACE] setupSchemaTables start")
	defer log.Println("[TRACE] setupSchemaTables end")

	// Iterate Tables & Build Modules
	for tableName, tableSchema := range schema.GetSchema() {
		// Translate Schema
		sc := getSQLiteColumnsFromTableSchema(tableSchema)

		current := NewModule(tableName, sc, tableSchema)
		if err := api.CreateModule(tableName, current, sqlite.ReadOnly(true)); err != nil {
			return err
		}
	}
	return nil
}
