// nozzle is a package for building your CloudFoundry(CF) nozzle.
// nozzle is a program which consume data from the Loggregator firehose
// (https://github.com/cloudfoundry/loggregator) and then select,
// buffer, and transform data and forward it to other applications,
// components or services.
//
// This pacakge provides the consumer which (1) gets the access token for
// firehose, (2) connects firehose and consume logs, (3) detects slow consumer alert.
// To get starts, see Config and Consumer.
//
// If you want to change the behavior of default consumer, then implement
// the interface of it.
package nozzle

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/cloudfoundry/noaa"
)

// By default, all logs goes to ioutil.Discard.
var defaultLogger = log.New(ioutil.Discard, "", log.LstdFlags)

// Config is a configuration struct for go-nozzle. It contains all required
// values for using this pacakge. This is used for argument when constructing
// nozzle client.
type Config struct {
	// DopplerAddr is a doppler firehose endpoint address to connect.
	// The address should start with 'wss://' (websocket endopint).
	DopplerAddr string

	// Token is an access token to connect to firehose. It's neccesary
	// to consume logs from doppler.
	//
	// If it's empty, the token is feched from UAA server.
	// To fetch token from UAA server, UaaAddr and Username/Password
	// for CF admin need to be set.
	Token string

	// SubscriptionID is unique id for a pool of clients of firehose.
	// For each SubscriptionID, all data will be distributed evenly
	// among that subscriber's client pool.
	SubscriptionID string

	// UaaAddr is UAA endpoint address. This is used for fetching access
	// token if Token is empty. To get token you also need to set
	// Username/Password for CloudFoundry admin.
	UaaAddr string

	// UaaTimeout is timeout to wait after sending request to uaa server.
	// The default value is 30 seconds.
	UaaTimeout time.Duration

	// Username is admin username of CloudFoundry. This is used for fetching
	// access token if Token is empty.
	Username string

	// Password is admin password of CloudFoundry. This is used for fetching
	// access token if Token is empty.
	Password string

	// Insecure is used for insecure connection with doppler.
	// Default value is false (Connect with TLS).
	Insecure bool

	// DebugPrinter is noaa.DebugPrinter. It's used for debugging
	// Noaa. Noaa is a client library to consume metric and log
	// messages from Doppler.
	DebugPrinter noaa.DebugPrinter

	// Logger is logger for go-nozzle. By default, output will be
	// discarded and not be displayed.
	Logger *log.Logger

	// The following fileds are now only for testing.
	tokenFetcher TokenFetcher
	rawConsumer  RawConsumer
}

// NewConsumer constructs a new consumer client for nozzle.
//
// You need access token for consuming firehose log. There is 2
// ways to construct. The one is to get token beforehand by yourself and use it.
// The other is to provide UAA endopoint with username/password for CloudFoundry
// admin to fetch the token.
//
// It returns error if the token is empty or can not fetch token from UAA
// If token is not empty or successfully getting from UAA, then it starts
// to consume firehose events and detecting slowConsumerAlerts.
func NewDefaultConsumer(config *Config) (Consumer, error) {
	if config.Logger == nil {
		config.Logger = defaultLogger
	}

	// If Token is not provided, fetch it by tokenFetcher.
	if config.Token != "" {
		config.Logger.Printf("[DEBUG] Using auth token (%s)",
			maskString(config.Token))
	} else {

		if config.UaaAddr == "" {
			return nil, fmt.Errorf("both Token and UaaAddr can not be empty")
		}

		fetcher := config.tokenFetcher
		if fetcher == nil {
			var err error
			fetcher, err = newDefaultTokenFetcher(config)
			if err != nil {
				return nil, fmt.Errorf("failed to construct default token fetcher: %s",
					err)
			}
		}

		// Execute tokenFetcher and get token
		token, err := fetcher.Fetch()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch token: %s", err)
		}

		config.Logger.Printf("[DEBUG] Setting auth token (%s)",
			maskString(token))
		config.Token = token
	}

	// Create new RawConsumer
	rc := config.rawConsumer
	if rc == nil {
		var err error
		rc, err = newRawConsumer(config)
		if err != nil {
			return nil, fmt.Errorf("failed to construct default consumer: %s", err)
		}
	}

	// Start consuming events from firehose.
	eventsCh, errCh := rc.Consume()

	// Construct default slowDetector
	sd := &defaultSlowDetector{
		logger: config.Logger,
	}

	// Start reading events from firehose and detect `slowConsumerAlert`.
	// The detection is notified by detectCh.
	eventsCh_, errCh_, detectCh := sd.Detect(eventsCh, errCh)

	return &consumer{
		rawConsumer:  rc,
		slowDetector: sd,
		eventCh:      eventsCh_,
		errCh:        errCh_,
		detectCh:     detectCh,
	}, nil
}

// maskString is used to mask string which should not be displayed
// directly like auth token
func maskString(s string) string {
	if len(s) < 10 {
		return "**** (masked)"
	}

	return s[:10] + "**** (masked)"
}
