# kafka-firehose-nozzle

`kafka-firehose-nozzle` is [CloudFoundry (CF) nozzle](https://docs.cloudfoundry.org/loggregator/architecture.html#nozzles) for [Apache Kafka](http://kafka.apache.org/). It consumes data from the Loggregator Firehose and then publishes it to Apache Kafka. 

The firehose generates events which are defined on [dropsonde-protocol](https://github.com/cloudfoundry/dropsonde-protocol). You can set Kafka topic for each event type (e.g., by default, `LogMessage` events are publish to `log-message` topic). Events are encoded in [protocol buffers](https://developers.google.com/protocol-buffers/) between CF components but when to publish to kafka, events are decoded to plain json text.

`kafka-firehose-nozzle` is written by Golang and built with [rakutentech/go-nozzle](https://github.com/rakutentech/go-nozzle) package. 

## Usage

Basic usage is,

```bash
$ kafak-firehose-nozzle [options]
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
