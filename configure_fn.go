package main

import (
	"errors"
	"fmt"

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
	var config string
	var err error
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
	pluginName := fmt.Sprintf("steampipe-plugin-%s", pluginAlias)

	c := &proto.ConnectionConfig{
		Connection:      pluginAlias,
		Plugin:          pluginName,
		PluginShortName: pluginAlias,
		Config:          config,
		PluginInstance:  pluginName,
	}

	if currentSchema != nil {
		// send an update request to the plugin server
		cs := []*proto.ConnectionConfig{c}
		req := &proto.UpdateConnectionConfigsRequest{Changed: cs}
		_, err := pluginServer.UpdateConnectionConfigs(req)
		if err != nil {
			return err
		}
	} else {
		// set the config in the plugin server
		cs := []*proto.ConnectionConfig{c}
		req := &proto.SetAllConnectionConfigsRequest{Configs: cs}
		_, err := pluginServer.SetAllConnectionConfigs(req)
		if err != nil {
			return err
		}
	}

	// we should also trigger a schema refresh after this call
	// reason we update is because we have already setup this plugin with a blank config
	// during bootstrap - so that we have all the tables setup
	// fetch the schema
	// we cannot use the global currentSchema variable here
	// because it may not have been loaded yet at all
	schema, err := getSchema()
	if err != nil {
		return err
	}

	if SCHEMA_MODE_DYNAMIC.Equals(schema.Mode) {
		// drop the existing tables - if they have been created
		if currentSchema != nil {
			conn := m.api.Connection()
			for tableName := range currentSchema.GetSchema() {
				if err := conn.Exec(fmt.Sprintf("DROP TABLE %s", tableName), nil); err != nil {
					return err
				}
			}
		}

		// create the tables for a dynamic schema
		if err := setupSchemaTables(schema, m.api); err != nil {
			return err
		}
		currentSchema = schema
	}

	return err
}

// getSchema returns the schema for the plugin
func getSchema() (*proto.Schema, error) {
	// Get Plugin Schema
	sRequest := &proto.GetSchemaRequest{Connection: pluginAlias}
	s, err := pluginServer.GetSchema(sRequest)
	if err != nil {
		return nil, err
	}
	return s.GetSchema(), nil
}

// setupSchemaTables sets up the schema tables for the plugin
// it fetched the schema from the plugin and then maps it to SQLite tables
func setupSchemaTables(schema *proto.Schema, api *sqlite.ExtensionApi) error {
	// Iterate Tables & Build Modules
	for tableName, tableSchema := range schema.GetSchema() {
		// Translate Schema
		sc, err := getSQLiteColumnsFromTableSchema(tableSchema)
		if err != nil {
			return err
		}

		current := NewModule(tableName, sc, tableSchema)
		if err := api.CreateModule(tableName, current, sqlite.ReadOnly(true)); err != nil {
			return err
		}
	}
	return nil
}
