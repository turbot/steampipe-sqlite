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
	if values[0].Type() != sqlite.SQLITE_TEXT {
		return "", (errors.New("expected a string argument"))
	}
	config = values[0].Text()
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

	// send an update request to the plugin server
	// we should also trigger a schema refresh after this call
	// reason we update is because we have already setup this plugin with a blank config
	// during bootstrap - so that we have all the tables setup
	cs := []*proto.ConnectionConfig{c}
	req := &proto.UpdateConnectionConfigsRequest{Changed: cs}
	_, err := pluginServer.UpdateConnectionConfigs(req)

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
