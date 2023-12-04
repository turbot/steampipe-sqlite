package main

import (
	"fmt"
	"strings"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

var currentSchema *proto.Schema
var schemaType = SCHEMA_MODE_STATIC

func register() {
	pluginServer.SetCacheOptions(&proto.SetCacheOptionsRequest{
		Enabled:   true,
		Ttl:       300,
		MaxSizeMb: 32,
	})

	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		configureFn := NewConfigureFn(api)
		fnName := fmt.Sprintf("steampipe_configure_%s", pluginAlias)
		fnName = strings.ToLower(fnName)
		if err := api.CreateFunction(fnName, configureFn); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		if SCHEMA_MODE_STATIC.Equals(pluginServer.GetSchemaMode()) {
			// if the target plugin has a static schema, then the list of tables and columns
			// is also static. let's just set it up with a blank config and setup the tables
			if err := setInitialConfig(); err != nil {
				return sqlite.SQLITE_ERROR, err
			}
			schema, err := getSchema()
			if err != nil {
				return sqlite.SQLITE_ERROR, err
			}
			if err := setupTables(schema, api); err != nil {
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
	req := &proto.SetAllConnectionConfigsRequest{
		Configs:        cs,
		MaxCacheSizeMb: 32,
	}

	_, err := pluginServer.SetAllConnectionConfigs(req)
	return err
}
