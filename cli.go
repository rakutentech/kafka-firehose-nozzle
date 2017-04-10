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
	"text/template"
	"time"

	"github.com/hashicorp/logutils"
	"github.com/rakutentech/go-nozzle"
	"golang.org/x/net/context"
)

//go:generate ./bin/kafka-firehose-nozzle -gen-godoc

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

const (
	// DefaultCfgPath is default config file path
	DefaultCfgPath = "example/kafka-firehose-nozzle.toml"

	// DefaultStatsInterval is default interval of displaying
	// stats info to console
	DefaultStatsInterval = 10 * time.Second

	// DefaultUsername to grant access token for firehose
	DefaultUsername = "admin"

	// DefaultUAATimeout is default timeout for requesting
	// auth token to UAA server.
	DefaultUAATimeout = 20 * time.Second

	// DefaultSubscriptionID is default subscription ID for
	// loggregagor firehose.
	DefaultSubscriptionID = "debug-kafka-firehose-nozzle"

	// DefaultIdleTimeout is the default timeout for receiving a single message
	// from the firehose
	DefaultIdleTimeout = 60 * time.Second
)

const (
	EnvPassword = "UAA_PASSWORD"
)

// godocFile is file name for godoc
const (
	godocFile = "doc.go"
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
		username       string
		password       string
		subscriptionID string
		logLevel       string

		worker int

		statsInterval time.Duration

		server   bool
		debug    bool
		version  bool
		genGodoc bool
	)

	// Define option flag parsing
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, helpText)
	}

	flags.StringVar(&cfgPath, "config", DefaultCfgPath, "")
	flags.StringVar(&subscriptionID, "subscription", "", "")
	flags.StringVar(&username, "username", "", "")
	flags.StringVar(&password, "password", os.Getenv(EnvPassword), "")
	flags.StringVar(&logLevel, "log-level", "INFO", "")
	flags.IntVar(&worker, "worker", runtime.NumCPU(), "")
	flags.DurationVar(&statsInterval, "stats-interval", DefaultStatsInterval, "")
	flags.BoolVar(&server, "server", false, "")
	flags.BoolVar(&debug, "debug", false, "")
	flags.BoolVar(&version, "version", false, "")

	// -gen-godoc flag is only for developers of this nozzle.
	// It generates godoc.
	flags.BoolVar(&genGodoc, "gen-godoc", false, "")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Generate godoc
	if genGodoc {
		if err := godoc(); err != nil {
			fmt.Fprintf(cli.errStream, "Faild to generate godoc %s\n", err)
			return ExitCodeError
		}

		fmt.Fprintf(cli.outStream, "Successfully generated godoc\n")
		return ExitCodeOK
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

	// Show basic infomation
	logger.Printf("[INFO] %s version: %s", Name, Version)
	logger.Printf("[INFO] Go version: %s (%s/%s)",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
	logger.Printf("[INFO] Num of CPU: %d", runtime.NumCPU())

	// Load configuration
	config, err := LoadConfig(cfgPath)
	if err != nil {
		logger.Printf("[ERROR] Failed to load configuration file: %s", err)
		return ExitCodeError
	}
	logger.Printf("[DEBUG] %#v", config)

	if subscriptionID != "" {
		config.SubscriptionID = subscriptionID
	} else if config.SubscriptionID == "" {
		config.SubscriptionID = DefaultSubscriptionID
	}

	if username != "" {
		config.CF.Username = username
	} else if config.CF.Username == "" {
		config.CF.Username = DefaultUsername
	}

	if password != "" {
		config.CF.Password = password
	}

	if config.CF.IdleTimeout == 0 {
		config.CF.IdleTimeout = int(DefaultIdleTimeout.Seconds())
	}

	// Initialize stats collector
	stats := NewStats()
	go stats.PerSec()

	// Start server.
	if server {
		Server := &Server{
			Logger: logger,
			Stats:  stats,
		}

		go Server.Start()
	}

	// Setup option struct for nozzle consumer.
	nozzleConfig := &nozzle.Config{
		DopplerAddr:    config.CF.DopplerAddr,
		Token:          config.CF.Token,
		UaaAddr:        config.CF.UAAAddr,
		Username:       config.CF.Username,
		Password:       config.CF.Password,
		IdleTimeout:    time.Duration(config.CF.IdleTimeout) * time.Second,
		SubscriptionID: config.SubscriptionID,
		Insecure:       config.InsecureSSLSkipVerify,
		Logger:         logger,
	}

	// Setup default nozzle consumer.
	nozzleConsumer, err := nozzle.NewConsumer(nozzleConfig)
	if err != nil {
		logger.Printf("[ERROR] Failed to construct nozzle consumer: %s", err)
		return ExitCodeError
	}

	err = nozzleConsumer.Start()
	if err != nil {
		logger.Printf("[ERROR] Failed to start nozzle consumer: %s", err)
		return ExitCodeError
	}
		
	// Setup nozzle producer
	var producer NozzleProducer
	if debug {
		logger.Printf("[INFO] Use LogProducer")
		producer = NewLogProducer(logger)
	} else {
		logger.Printf("[INFO] Use KafkaProducer")
		var err error
		producer, err = NewKafkaProducer(logger, stats, config)
		if err != nil {
			logger.Printf("[ERROR] Failed to construct kafka producer: %s", err)
			return ExitCodeError
		}
	}

	// Create a ctx for cancelation signal across the goroutined producers.
	ctx, cancel := context.WithCancel(context.Background())

	// Display stats in every x seconds.
	go func() {
		logger.Printf("[INFO] Stats display interval: %s", statsInterval)
		ticker := time.NewTicker(statsInterval)
		for {
			select {
			case <-ticker.C:
				logger.Printf("[INFO] Consume per sec: %d", stats.ConsumePerSec)
				logger.Printf("[INFO] Consumed messages: %d", stats.Consume)

				logger.Printf("[INFO] Publish per sec: %d", stats.PublishPerSec)
				logger.Printf("[INFO] Published messages: %d", stats.Publish)

				logger.Printf("[INFO] Publish delay: %d", stats.Consume-stats.Publish-stats.PublishFail)

				logger.Printf("[INFO] SubInput buffer: %d", stats.SubInputBuffer)

				logger.Printf("[INFO] Failed consume: %d", stats.ConsumeFail)
				logger.Printf("[INFO] Failed publish: %d", stats.PublishFail)
				logger.Printf("[INFO] SlowConsumer alerts: %d", stats.SlowConsumerAlert)
			}
		}
	}()

	// Handle nozzle consumer error and slow consumer alerts
	go func() {
		for {
			select {
			// The following comments are from noaa client comments.
			//
			// "Whenever an error is encountered, the error will be sent down the error
			// channel and Firehose will attempt to reconnect up to 5 times.  After five
			// failed reconnection attempts, Firehose will give up and close the error and
			// Envelope channels."
			//
			// When noaa client gives up recconection to doppler, nozzle should be
			// terminated instead of infinity loop. We deploy nozzle on CF as CF app.
			// This means we can ask platform to restart nozzle when terminated.
			//
			// In future, we should implement restart/retry fuctionality in nozzle (or go-nozzle)
			// and avoid to rely on the specific platform (so that we can deploy this anywhere).
			case err, ok := <-nozzleConsumer.Errors():

				// This means noaa consumer stopped consuming and close its channel.
				if !ok {
					logger.Printf("[ERROR] Nozzle consumer's error channel is closed")

					// Call cancellFunc and then stop all nozzle workers
					cancel()

					// Finish error handline goroutine
					return
				}

				// Connection retry is done on noaa side (5 times)
				// After 5 times but can not be recovered, then channel is closed.
				logger.Printf("[ERROR] Received error from nozzle consumer: %s", err)
				stats.Inc(ConsumeFail)

			case err := <-nozzleConsumer.Detects():
				// TODO(tcnksm): Should know how many logs are dropped.
				logger.Printf("[ERROR] Detect slowConsumerAlert: %s", err)
				stats.Inc(SlowConsumerAlert)
			}
		}
	}()

	// Now we don't use this. But in future, it hould be used
	// for monitoring
	go func() {
		for _ = range producer.Successes() {
			stats.Inc(Publish)
		}
	}()

	// Handle producer error
	// TODO(tcnksm): Buffer and restart when it recovers
	go func() {
		// cancel all other producer goroutine
		defer cancel()

		for err := range producer.Errors() {
			logger.Printf("[ERROR] Faield to produce logs: %s", err)
			stats.Inc(PublishFail)
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

func godoc() error {
	f, err := os.Create(godocFile)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, err := template.New("godoc").Parse(godocTmpl)
	if err != nil {
		return err
	}
	return tmpl.Execute(f, helpText)
}

var godocTmpl = `// THIS FILE IS GENERATED BY GO GENERATE.
// DO NOT EDIT THIS FILE BY HAND.

/*
{{ . }}
*/
package main
`

// helpText is used for flag usage messages.
var helpText = `kafka-firehose-nozzle is Cloud Foundry nozzle which forwards logs from
the loggeregagor firehose to Apache kafka (http://kafka.apache.org/).

Usage:

    kafak-firehose-nozzle [options]

Available options:

    -config PATH          Path to configuraiton file    
    -username NAME        username to grant access token to connect firehose
    -password PASS        password to grant access token to connect firehose
    -worker NUM           Number of producer worker. Default is number of CPU core
    -subscription ID      Subscription ID for firehose. Default is 'kafka-firehose-nozzle'
    -stats-interval TIME  How often display stats info to console  
    -debug                Output event to stdout instead of producing message to kafka
    -log-level LEVEL      Log level. Default level is INFO (DEBUG|INFO|ERROR)
`
