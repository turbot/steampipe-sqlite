package main

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"

	"github.com/turbot/steampipe-plugin-aws/aws"
)

var pluginServer = plugin.Server(&plugin.ServeOpts{PluginFunc: aws.Plugin})
var pluginAlias = "aws"

func init() {
	setupLogger(pluginAlias)
	register()
}

func main() {
	// DO NOT USE
  // this code base is meant to be compiled as a dynamically linked library
  // the main function only exists to satisfy the go compiler
}
