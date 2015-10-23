package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"runtime"

	"github.com/Shopify/sarama"
	"github.com/eapache/go-resiliency/breaker"
	"github.com/hashicorp/logutils"
	"github.com/rakutentech/go-nozzle"
)

const (
	// DefaultCfgPath is default config file path
	DefaultCfgPath = "./config/kafka-firehose-nozzle.toml"

	// DefaultUAATimeout is default timeout for requesting
	// auth token to UAA server.
	DefaultUAATimeout = 20 * time.Second

	// DefaultSubscriptionID is default subscription ID for
	// loggregagor firehose
	DefaultSubscriptionID = "kafka-firehose-nozzle"
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
		worker         int
		varz           bool
		insecure       bool
		logLevel       string
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
	flags.BoolVar(&insecure, "insecure", false, "")
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

	// Load configuration
	config, err := LoadConfig(cfgPath)
	if err != nil {
		logger.Printf("[ERROR] Failed to load configuration file: %s", err)
		return ExitCodeError
	}

	// Start varz server.
	// This is for running this app as PaaS application.
	if varz {
		varzServer := &VarzServer{Logger: logger}
		go varzServer.Start()
	}

	// Setup kafka async producer.
	// Should be configurable more from .toml file.
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Retry.Max = 5
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll

	asyncProducer, err := sarama.NewAsyncProducer(config.Kafka.Brokers, producerConfig)
	if err != nil {
		logger.Printf("[ERROR] Failed to construct kafka producer: %s", err)
		return ExitCodeError
	}

	producer := &KafkaProducer{
		AsyncProducer: asyncProducer,
		DoneCh:        make(chan struct{}, 1),
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
		logger.Printf("[ERROR] Failed to construct nozzle: %s", err)
		return ExitCodeError
	}

	// Handle signal of interrupting to stop process safely.
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-producer.DoneCh:
			// Do nothing
		case <-signalCh:
			logger.Println("[INFO] Interrupt Received: closing connection with firehose")
			close(producer.DoneCh)
		}
	}()

	// Handle error from consumer & producer
	go func() {
		for {
			select {
			case <-producer.DoneCh:
				// Stop immediately.
				// Without this we closed chan is closed and panic..
				return
			case err := <-nozzleConsumer.Errors():
				if err == nil {
					continue
				}
				// Connection retry is done on noaa side (5 times)
				// TODO: Check retry is done before returns error in noaa.
				// See datadog-firehose-nozzle way to do this.
				logger.Printf("[ERROR] Received error from nozzle consumer: %s", err)

				// Stop producer process
				close(producer.DoneCh)
				return
			case err := <-producer.Errors():
				if err == nil {
					continue
				}
				// TODO: Need to disscuss (same as the case slowConsumerAlerts).
				// This is critical error, we need to know as fast as possible.
				logger.Printf("[ERROR] Faield to produce nozzle logs: %s", err)

				if err.Err != breaker.ErrBreakerOpen {
					continue
				}

				// If circuit breaker is open, we assume recover is difficult.
				// So Stop produceer process.
				logger.Printf("[ERROR] Faield to recover kafka producer: ", err.Err)
				close(producer.DoneCh)
				return
			}
		}
	}()

	// Handle slowConsumer alerts. Now only notify by logger.
	go func() {
		for _ = range nozzleConsumer.Detects() {
			// TODO: Need to discuss (same as the case kafka producer).
			// This is critical error, we need to know as fast as possible.
			logger.Printf("[ERROR] Detect slowConsumerAlert")
		}
	}()

	// Start multiple produce worker process.
	var wg sync.WaitGroup
	logger.Printf("[INFO] Start %d producer process", worker)
	for i := 0; i < worker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			producer.produce(nozzleConsumer.Events())
		}()
	}

	// Wait until all producer process is done.
	wg.Wait()

	// Attempt to close all the things. Not retuns soon even
	// if error is happend while closing.
	isError := false

	// Close nozzle consumer
	logger.Printf("[INFO] Closing nozzle cosumer")
	if err := nozzleConsumer.Close(); err != nil {
		logger.Printf("[ERROR] Failed to close nozzle consumer process: %s", err)
		isError = true
	}

	logger.Printf("[INFO] Closing kafka producer")
	if err := producer.Close(); err != nil {
		logger.Printf("[ERROR] Failed to close kafka producer: %s", err)
		isError = true
	}

	logger.Printf("[INFO] Finished kafka firehose nozzle")
	if isError {
		return ExitCodeError
	}
	return ExitCodeOK
}

// helpText is used for flag usage messages.
var helpText = `kafka-firehose-nozzle is a component which forwards logs from
the loggeregagor firehost to Apache kafka.

Usage

    kafak-firehose-nozzle [options]

Avairalbe options

    -config PATH       Path to configuraiton file.
    -worker NUM        Number of producer worker. Default is number of CPU core.
    -subscription ID   Subscription ID for firehose. Default is 'kafka-firehose-nozzle'.
    -insecure          Enable insecure connection.
    -log-level LEVEL   Log level. Default level is INFO (DEBUG|INFO|ERROR).
`
