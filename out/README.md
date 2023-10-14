# Steampipe SQLite Extension aws

## Prerequisites
- A build of SQLite that supports extensions (default `brew` install has extensions disabled).

## Configuration
If you require [configuration](https://hub.steampipe.io/plugins/turbot/aws#configuration) for the extension, you need to set this prior to loading the extension.

- Create the configuration table if it doesn't exist:
  - `CREATE TABLE IF NOT EXISTS aws_config(config TEXT);`
- Insert your configuration into the config table:
  - `INSERT INTO aws_config(config) VALUES('your_config="GoesHere"')`

## Installation
- Copy the binary `steampipe-sqlite-extension-aws.so` to a directory of choice.
- Start your `sqlite` instance.
  - `sqlite3` or `sqlite3 myDB.file`
- Load the extension into `sqlite`. 
  - `.load /path/to/steampipe-sqlite-extension-aws.so`

## Usage
Please refer to the [Table Documentation](https://hub.steampipe.io/plugins/turbot/aws/tables).