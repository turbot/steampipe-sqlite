package main

import (
	"fmt"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

type CreateVirtualTablesSqliteFunction struct{}

func register() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		// set a blank config, so that we can fetch the schema from the plugin
		if err := setInitialConfig(); err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		schema, err := getSchema()
		if err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		if err := setupSchemaTables(schema, api); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		configureFn := NewConfigureFn(api)
		if err := api.CreateFunction(fmt.Sprintf("configure_%s", pluginAlias), configureFn); err != nil {
			return sqlite.SQLITE_ERROR, err
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
