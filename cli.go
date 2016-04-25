package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/logutils"
	"github.com/rakutentech/go-nozzle"
	"golang.org/x/net/context"
)

const (
	// DefaultCfgPath is default config file path
	DefaultCfgPath = "kafka-firehose-nozzle.toml"

	// DefaultUAATimeout is default timeout for requesting
	// auth token to UAA server.
	DefaultUAATimeout = 20 * time.Second

	// DefaultSubscriptionID is default subscription ID for
	// loggregagor firehose
	DefaultSubscriptionID = "debug-kafka-firehose-nozzle"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		cfgPath        string
		subscriptionID string
		logLevel       string
		worker         int
		varz           bool
		debug          bool
		version        bool
	)

	// Define option flag parsing
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, helpText)
	}

	flags.StringVar(&cfgPath, "config", DefaultCfgPath, "")
	flags.StringVar(&subscriptionID, "subscription", DefaultSubscriptionID, "")
	flags.StringVar(&logLevel, "log-level", "INFO", "")
	flags.IntVar(&worker, "worker", runtime.NumCPU(), "")
	flags.BoolVar(&varz, "varz-server", false, "")
	flags.BoolVar(&debug, "debug", false, "")
	flags.BoolVar(&version, "version", false, "")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	// Setup logger with level Filtering
	logger := log.New(&logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "ERROR"},
		MinLevel: (logutils.LogLevel)(strings.ToUpper(logLevel)),
		Writer:   cli.outStream,
	}, "", log.LstdFlags)

	logger.Printf("[INFO] LogLevel: %s", logLevel)
	logger.Printf("[INFO] Subscription ID: %s", subscriptionID)

	// Load configuration
	config, err := LoadConfig(cfgPath)
	if err != nil {
		logger.Printf("[ERROR] Failed to load configuration file: %s", err)
		return ExitCodeError
	}

	// Start varz server.
	// This is for running this app as PaaS application (need to accept http request)
	if varz {
		varzServer := &VarzServer{Logger: logger}
		go varzServer.Start()
	}

	// Setup option struct for nozzle consumer.
	nozzleConfig := &nozzle.Config{
		DopplerAddr:    config.CF.DopplerAddr,
		UaaAddr:        config.CF.UAAAddr,
		Username:       config.CF.Username,
		Password:       config.CF.Password,
		SubscriptionID: subscriptionID,
		Logger:         logger,
	}

	// Setup default nozzle consumer.
	nozzleConsumer, err := nozzle.NewDefaultConsumer(nozzleConfig)
	if err != nil {
		logger.Printf("[ERROR] Failed to construct nozzle consumer: %s", err)
		return ExitCodeError
	}

	var producer NozzleProducer
	if debug {
		logger.Printf("[INFO] Use LogProducer")
		producer = NewLogProducer(logger)
	} else {
		logger.Printf("[INFO] Use KafkaProducer")
		var err error
		producer, err = NewKafkaProducer(config)
		if err != nil {
			logger.Printf("[ERROR] Failed to construct kafka producer: %s", err)
			return ExitCodeError
		}
	}

	// Create a ctx for cancelation signal across the goroutined producers.
	ctx, cancel := context.WithCancel(context.Background())

	// Handle nozzle consumer error and slow consumer alerts
	go func() {
		for {
			select {
			case err := <-nozzleConsumer.Errors():
				if err == nil {
					continue
				}

				// Connection retry is done on noaa side (5 times)
				logger.Printf("[ERROR] Received error from nozzle consumer: %s", err)

			case err := <-nozzleConsumer.Detects():
				logger.Printf("[ERROR] Detect slowConsumerAlert: %s", err)
			}
		}
	}()

	// Handle producer error
	go func() {
		// cancel all other producer goroutine
		defer cancel()

		for err := range producer.Errors() {
			if err == nil {
				continue
			}

			logger.Printf("[ERROR] Faield to produce logs: %s", err)
			return
		}
	}()

	// Handle signal of interrupting to stop process safely.
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt, os.Kill)
	go func() {
		<-signalCh
		logger.Println("[INFO] Interrupt Received: cancel all producers")
		cancel()
	}()

	// Start multiple produce worker processes.
	// nozzle consumer events will be distributed to each producer.
	// And each producer produces message to kafka.
	//
	// Process will be blocked until all producer process finishes each jobs.
	var wg sync.WaitGroup
	logger.Printf("[INFO] Start %d producer process", worker)
	for i := 0; i < worker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			producer.Produce(ctx, nozzleConsumer.Events())
		}()
	}

	// Wait until all producer process is done.
	wg.Wait()

	// Attempt to close all the things. Not returns soon even if
	// error is happend while closing.
	isError := false

	// Close nozzle consumer
	logger.Printf("[INFO] Closing nozzle cosumer")
	if err := nozzleConsumer.Close(); err != nil {
		logger.Printf("[ERROR] Failed to close nozzle consumer process: %s", err)
		isError = true
	}

	logger.Printf("[INFO] Closing producer")
	if err := producer.Close(); err != nil {
		logger.Printf("[ERROR] Failed to close producer: %s", err)
		isError = true
	}

	logger.Printf("[INFO] Finished kafka firehose nozzle")
	if isError {
		return ExitCodeError
	}
	return ExitCodeOK
}

// helpText is used for flag usage messages.
var helpText = `kafka-firehose-nozzle is a tool to forward logs from
the loggeregagor firehose to Apache kafka.

Usage

    kafak-firehose-nozzle [options]

Avairalbe options

    -config PATH       Path to configuraiton file
    -worker NUM        Number of producer worker. Default is number of CPU core
    -subscription ID   Subscription ID for firehose. Default is 'kafka-firehose-nozzle'
    -debug             Output event to stdout instead of producing message to kafka
    -log-level LEVEL   Log level. Default level is INFO (DEBUG|INFO|ERROR)
`
