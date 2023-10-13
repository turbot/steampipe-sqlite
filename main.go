package main

import (
	"fmt"
	"github.com/turbot/steampipe-plugin-aws/aws"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"go.riyazali.net/sqlite"
)

var pluginServer = plugin.Server(&plugin.ServeOpts{PluginFunc: aws.Plugin})
var pluginAlias = "aws"

func init() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		// NOTE: Need user to pre-populate config into a specific table...
		var config string
		conn := api.Connection()
		config, err := getPluginConfig(conn, pluginAlias)
		if err != nil {
			config = ""
		}

		// Set Connection Config
		err = setPluginConnectionConfig(pluginAlias, config)
		if err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		// Get Plugin Schema
		sRequest := &proto.GetSchemaRequest{Connection: pluginAlias}
		s, err := pluginServer.GetSchema(sRequest)
		if err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		// Iterate Tables & Build Modules
		for tableName, schema := range s.Schema.Schema {
			// Translate Schema
			sc, err := parsePluginSchema(schema)
			if err != nil {
				return sqlite.SQLITE_ERROR, err
			}

			current := NewModule(tableName, sc, schema)
			if err := api.CreateModule(tableName, current); err != nil {
				return sqlite.SQLITE_ERROR, err
			}
		}
		return sqlite.SQLITE_OK, nil
	})
}

func main() {
	// DO NOT USE
}

func getPluginConfig(conn *sqlite.Conn, alias string) (string, error) {
	configTable := fmt.Sprintf("%s_config", alias)
	query := fmt.Sprintf("SELECT config FROM %s LIMIT 1;", configTable)
	var config string

	err := conn.Exec(query, func(stmt *sqlite.Stmt) error {
		config = stmt.GetText("config")
		return nil
	})
	if err != nil {
		return "", err
	}

	return config, nil
}

func setPluginConnectionConfig(alias string, config string) error {
	pName := fmt.Sprintf("steampipe-plugin-%s", alias) // TODO: grab ful from ociimage

	c := &proto.ConnectionConfig{
		Connection:      alias,
		Plugin:          pName,
		PluginShortName: alias,
		Config:          config,
		PluginInstance:  pName,
	}

	cs := []*proto.ConnectionConfig{c}
	req := &proto.SetAllConnectionConfigsRequest{Configs: cs}

	_, err := pluginServer.SetAllConnectionConfigs(req)
	return err
}
