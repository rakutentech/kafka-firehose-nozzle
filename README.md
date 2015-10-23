# kafka-firehose-nozzle

kafka-firehose-nozzle is a component which forwards logs from the loggeregagor firehose to Apache kafka.


## Usage

```bash
$ kafka-firehose-nozzle -config kafka-firehose-nozzle.toml
```

## Build 

```bash
$ go get -d github.com/rakutentech/kafka-firehose-nozzle
$ cd $GOPATH/src/github.com/rakutentech/kafka-firehose-nozzle
$ make build
```

## Contribution

1. Fork ([https://github.com/rakutentech/kafka-firehose-nozzle/fork](https://github.com/rakutentech/kafka-firehose-nozzle/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `make test-all` command and confirm that it passes
1. Create a new Pull Request

## Author

[Taichi Nakashima](https://github.com/tcnksm)
