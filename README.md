# kafka-firehose-nozzle

[![Build Status](http://img.shields.io/travis/rakutentech/kafka-firehose-nozzle.svg?style=flat-square)](https://travis-ci.org/rakutentech/kafka-firehose-nozzle) [![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/rakutentech/kafka-firehose-nozzle)

`kafka-firehose-nozzle` is a [Cloud Foundry firehose nozzle](https://docs.cloudfoundry.org/loggregator/architecture.html#nozzles) for [Apache Kafka](http://kafka.apache.org/). It consumes events from the Loggregator Firehose and publishes it to Apache Kafka. It is used in production in the Rakuten Platform-as-a-Service where it readily handles peaks of 30K events/second.

The Cloud Foundry firehose generates protobuf-encoded events as defined in the [dropsonde-protocol](https://github.com/cloudfoundry/dropsonde-protocol).
kafka-firehose-nozzle publishes JSON-encoded events to Kafka and allows to specify, for each event type (e.g LogMessage, ContainerMetric, ...), the topic to use.

kafka-firehose-nozzle is stateless, requires no external databases and can easily
scale out for additional capacity. It can also be deployed as a Cloud Foundry application.

`kafka-firehose-nozzle` is written by Golang and built with [rakutentech/go-nozzle](https://github.com/rakutentech/go-nozzle) package.

## Usage

Basic usage is,

```bash
$ kafka-firehose-nozzle [options]
```

The following are available options,

```bash
-config PATH       Path to configuraiton file
-username NAME     username to grant access token to connect firehose
-password PASS     password to grant access token to connect firehose
-worker NUM        Number of producer worker. Default is number of CPU core
-subscription ID   Subscription ID for firehose. Default is 'kafka-firehose-nozzle'
-debug             Output event to stdout instead of producing message to kafka
-log-level LEVEL   Log level. Default level is INFO (DEBUG|INFO|ERROR)
```

You can set `password` via `UAA_PASSWORD` environmental variable.

## Configuration

You can configure it via `.toml` file. You can see the example and description of this configuration file in [example](/example) directory.

## Install

To install, you can use `go get` command,

```bash
$ go get github.com/rakutentech/kafka-firehose-nozzle
```

You can deploy this as Cloud Foundry application with [go-buildpack](https://github.com/cloudfoundry/go-buildpack).

## Contribution

1. Fork ([https://github.com/rakutentech/kafka-firehose-nozzle/fork](https://github.com/rakutentech/kafka-firehose-nozzle/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `make test-all` command and confirm that it passes
1. Create a new Pull Request

## Author

[Taichi Nakashima](https://github.com/tcnksm)
