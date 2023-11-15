package main

import (
	"fmt"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

var currentSchema *proto.Schema
var schemaType = SCHEMA_MODE_STATIC

func register() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		configureFn := NewConfigureFn(api)
		if err := api.CreateFunction(fmt.Sprintf("configure_%s", pluginAlias), configureFn); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		if err := setInitialConfig(); err != nil {
			// if this errors, don't try to fetch the schema
			// assume that this is a dynamic plugin and that we
			// cannot just set a blank config
			return sqlite.SQLITE_OK, nil
		}
		schema, err := getSchema()
		if err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		if SCHEMA_MODE_STATIC.Equals(schema.Mode) {
			// create the tables for a plugin with static schema
			if err := setupSchemaTables(schema, api); err != nil {
				return sqlite.SQLITE_ERROR, err
			}
			currentSchema = schema
		}

		return sqlite.SQLITE_OK, nil
	})
}

func setInitialConfig() error {
	pluginName := fmt.Sprintf("steampipe-plugin-%s", pluginAlias)

	c := &proto.ConnectionConfig{
		Connection:      pluginAlias,
		Plugin:          pluginName,
		PluginShortName: pluginAlias,
		PluginInstance:  pluginName,
		// set a blank config, so that we can fetch the schema from the plugin
		Config: "",
	}

	cs := []*proto.ConnectionConfig{c}
	req := &proto.SetAllConnectionConfigsRequest{Configs: cs}

	_, err := pluginServer.SetAllConnectionConfigs(req)
	return err
}
