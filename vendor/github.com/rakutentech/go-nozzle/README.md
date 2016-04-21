go-nozzle [![Build Status](http://img.shields.io/travis/rakutentech/go-nozzle.svg?style=flat-square)](https://travis-ci.org/rakutentech/go-nozzle) [![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/rakutentech/go-nozzle) 
====

`go-nozzle` is a pacakge for Go (golang) for building [CloudFoundry(CF) nozzle](https://docs.cloudfoundry.org/loggregator/architecture.html#nozzles). Nozzle is a program which consume data from the Loggregator Firehose and then select, buffer, and transform data, and forward it to other applications like [Apache Kafka](http://kafka.apache.org/) or external services like [Data Dog](https://www.datadoghq.com/).

With this package, you can create a nozzle client which gets the access token for firehose from UAA (when token is not provided), connects firehose endpoint and consumes logs (by default, it uses [cloudfoundry/noaa](github.com/cloudfoundry/noaa) client) and detects slow consumer alert. All you need to do is writing output part of nozzle (e.g., kafka producing). 

Documentation is available on [GoDoc](http://godoc.org/github.com/rakutentech/go-nozzle).

## Install

To install, use `go get`:

```bash
$ go get github.com/rakutentech/go-nozzle
```

## Usage

The following is the simple example, 


```golang
// Setup consumer configuration
config := &nozzle.Config{
	DopplerAddr:    "wss://doppler.cloudfoundry.net",
    UaaAddr:        "https://uaa.cloudfoundry.net",
    Username:       "tcnksm",
    Password:       "xyz",
    SubscriptionID: "go-nozzle-example-A",
}
   
// Create default consumer
consumer, _  := nozzle.NewDefaultConsumer(config)

// Consume events
event := <-consumer.Events()
... 
```


Also you can check the example usage of `go-nozzle` on [example](/example) directory. 


## Author

[Taichi Nakashima](https://github.com/tcnksm)
