<p align="center">
    <h1 align="center">Steampipe SQLite Extension</h1>
</p>

<p align="center">
  <a aria-label="Steampipe logo" href="https://steampipe.io">
    <img src="https://steampipe.io/images/steampipe_logo_wordmark_padding.svg" height="28">
  </a>
</p>

## Overview

The `Steampipe SQLite Extension library` is a `SQLite3` extension that wraps [Steampipe](https://steampipe.io) plugins to interface with SQLite. 

See the [Writing Plugins](https://steampipe.io/docs/develop/writing-plugins) guide to get started writing Steampipe plugins.

## Building an extension

If you want to build the extension for a single plugin, follow these steps. This process allows you to build the SQLite extension specifically for one particular plugin, rather than for Steampipe.

Make sure that you have the following installed in your system:
1. `go`
1. `make` (you will also need to ensure `CGO` is enabled)

Steps:
1. Clone this repository onto your system
1. Change to the cloned directory
1. Run the following commands:
```shell
make build plugin="<plugin short name>" plugin_github_url="<plugin repo github URL>"
```
Replace <plugin short name> with the alias or short name of your plugin and <plugin repo GitHub URL> with the GitHub URL of the plugin's repository.

This command will compile an extension specifically for the chosen plugin generating a binary `steampipe-sqlite-extension-<plugin short name>.so`.

This can be loaded into your `sqlite` instance using a command similar to:
```shell
.load /path/to/steampipe-sqlite-extension-<plugin short name>.so
```

### Example:

In order to build an extension wrapping the `AWS` [plugin](https://github.com/turbot/steampipe-plugin-aws). You would run the following command:
```shell
make build plugin_alias="aws" plugin_github_url="github.com/turbot/steampipe-plugin-aws"
```

## Get involved

### Community

The Steampipe community can be found on [Slack](https://turbot.com/community/join), where you can ask questions, voice ideas, and share your projects.

Our [Code of Conduct](https://github.com/turbot/steampipe/blob/main/CODE_OF_CONDUCT.md) applies to all Steampipe community channels.

### Contributing

Please see [CONTRIBUTING.md](https://github.com/turbot/steampipe/blob/main/CONTRIBUTING.md).

