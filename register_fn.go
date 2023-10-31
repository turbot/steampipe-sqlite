package main

import (
	"errors"
	"fmt"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
)

// CreateVTab implements a custom CreateVTab(...) scalar sql function
type CreateVTab struct {
	api *sqlite.ExtensionApi
}

func NewCreateVTab(api *sqlite.ExtensionApi) *CreateVTab {
	return &CreateVTab{
		api: api,
	}
}

func (m *CreateVTab) Args() int           { return 1 }
func (m *CreateVTab) Deterministic() bool { return true }
func (m *CreateVTab) getConfig(values ...sqlite.Value) (config string, err error) {
	if len(values) > 1 {
		return "", errors.New("expected a single argument")
	}
	if values[0].Type() != sqlite.SQLITE_TEXT {
		return "", (errors.New("expected a string argument"))
	}
	config = values[0].Text()
	return config, nil
}

func (m *CreateVTab) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	var config string
	var err error
	if config, err = m.getConfig(values...); err != nil {
		ctx.ResultError(err)
		return
	}
	// Set Connection Config
	err = m.setPluginConnectionConfig(config)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	// Get Plugin Schema
	sRequest := &proto.GetSchemaRequest{Connection: pluginAlias}
	s, err := pluginServer.GetSchema(sRequest)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	// Iterate Tables & Build Modules
	for tableName, schema := range s.Schema.Schema {
		// Translate Schema
		sc, err := parsePluginSchema(schema)
		if err != nil {
			ctx.ResultError(err)
			return
		}

		current := NewModule(tableName, sc, schema)
		if err := m.api.CreateModule(tableName, current); err != nil {
			ctx.ResultError(err)
			return
		}
	}

	ctx.ResultText("")
}

func (m *CreateVTab) setPluginConnectionConfig(config string) error {
	pName := fmt.Sprintf("steampipe-plugin-%s", pluginAlias) // TODO: grab ful from ociimage

	c := &proto.ConnectionConfig{
		Connection:      pluginAlias,
		Plugin:          pName,
		PluginShortName: pluginAlias,
		Config:          config,
		PluginInstance:  pName,
	}

	cs := []*proto.ConnectionConfig{c}
	req := &proto.SetAllConnectionConfigsRequest{Configs: cs}

	_, err := pluginServer.SetAllConnectionConfigs(req)
	return err
}
