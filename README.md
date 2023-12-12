# Steampipe SQLite

A family of SQLite extensions, each derived from a [Steampipe plugin](https://hub.steampipe.io/plugins), that fetch data from cloud services and APIs.

## Getting Started

You can use an installer that enables you to choose a plugin and download the SQLite extension for that plugin.

**[Installation guide →](https://steampipe.io/docs/steampipe_sqlite/install)**

## Examples

### Select EC2 instances

```sql
select * from aws_ec2_instance;
```

### Filter to running instances

```sql
select * from aws_ec2_instance
where instance_state='running';
```

### Select a subset of columns

```sql
select arn, instance_state from aws_ec2_instance
where instance_state='running';
```

### Limit results

```sql
select arn, instance_state from aws_ec2_instance
where instance_state='running'
limit 10;
```
## Developing

To build an extension, use the provided `Makefile`. For example, to build the AWS extension, run the following command. The built extension lands in your current directory. 

```bash
make build plugin=aws
```

## Open Source & Contributing

This repository is published under the [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0) license. Please see our [code of conduct](https://github.com/turbot/.github/blob/main/CODE_OF_CONDUCT.md). We look forward to collaborating with you!

[Steampipe](https://steampipe.io) is a product produced exclusively by [Turbot HQ, Inc](https://turbot.com). It is distributed under our commercial terms. Others are allowed to make their own distribution of the software, but cannot use any of the Turbot trademarks, cloud services, etc. You can learn more in our [Open Source FAQ](https://turbot.com/open-source).

