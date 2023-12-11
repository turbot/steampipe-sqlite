# Steampipe SQLite

A family of SQLite extensions, each derived from a [Steampipe plugin](https://hub.steampipe.io/plugins), that fetch data from cloud services and APIs.

## Getting Started

You can use an installer that enables you to choose a plugin and download the SQLite extension for that plugin. See the [installation docs](https://turbot.com/docs/steampipe_export/install) for details. 


## Examples

### Select EC2 instances

```bash
select * from aws_ec2_instance;
```

### Filter to running instances

```bash
select * from aws_ec2_instance
where instance_state='running';
```

### Select a subset of columns

```bash
select arn, instance_state from aws_ec2_instance
where instance_state='running';
```

### Limit results

```bash
select arn, instance_state from aws_ec2_instance
where instance_state='running'
limit 10;
```
## Developing

To build an extension, use the provided `Makefile`. For example, to build the AWS extension, run the following command. The built extension lands in your current directory. 

```bash
make build plugin=aws
```

## Prerequisites

- [Golang](https://golang.org/doc/install) Version 1.21 or higher.

## Contributing
If you would like to contribute to this project, please open an issue or create a pull request. We welcome any improvements or bug fixes. Contributions are subject to the [Apache-2.0](https://opensource.org/license/apache-2-0/) license.

