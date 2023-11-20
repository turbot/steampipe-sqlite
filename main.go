package main

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc"
	_ "github.com/turbot/steampipe-plugin-sdk/v5/logging"
)

var pluginServer *grpc.PluginServer
var pluginAlias string

func main() {
	// this main file will be overwritten when the template at ./templates/main.go.tmpl is rendered
	//
	// we keep this file here to ensure that the plugin can be built without the template
	// this allows us to have a compiling go module that dependency trackers can use
	//
}
