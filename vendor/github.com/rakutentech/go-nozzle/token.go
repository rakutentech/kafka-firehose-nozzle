package nozzle

import (
	"fmt"
	"log"
	"time"

	"github.com/cloudfoundry-incubator/uaago"
)

const (
	defaultUAATimeout = 30 * time.Second
)

// TokenFetcher is the interface for fetching access token
// From UAA server. By default, defaultTokenFetcher
// (which is implemented with https://github.com/cloudfoundry-incubator/uaago)
// is used
type TokenFetcher interface {
	// Fetch fetches the token from Uaa and return it. If any, returns error.
	Fetch() (string, error)
}

type defaultTokenFetcher struct {
	uaaAddr  string
	username string
	password string
	timeout  time.Duration

	logger *log.Logger
}

// Fetch gets access token from UAA server. This auth token
// is s used for accessing traffic-controller. It retuns error if any.
func (tf *defaultTokenFetcher) Fetch() (string, error) {
	tf.logger.Printf("[INFO] Getting auth token of %q from UAA (%s)", tf.username, tf.uaaAddr)
	client, err := uaago.NewClient(tf.uaaAddr)
	if err != nil {
		return "", err
	}

	resCh, errCh := make(chan string), make(chan error)
	go func() {
		token, err := client.GetAuthToken(tf.username, tf.password, false)
		if err != nil {
			errCh <- err
		}
		resCh <- token
	}()

	timeout := defaultUAATimeout
	if tf.timeout != 0 {
		timeout = tf.timeout
	}

	select {
	case err := <-errCh:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("request timeout: %s", timeout)
	case token := <-resCh:
		return token, nil
	}
}

func (tf *defaultTokenFetcher) validate() error {

	if tf.uaaAddr == "" {
		return fmt.Errorf("UaaAddr must not be empty")
	}

	if tf.username == "" {
		return fmt.Errorf("Username must not be empty")
	}

	if tf.password == "" {
		return fmt.Errorf("Password must not be empty")
	}

	return nil
}

func newDefaultTokenFetcher(config *Config) (*defaultTokenFetcher, error) {
	fetcher := &defaultTokenFetcher{
		uaaAddr:  config.UaaAddr,
		timeout:  config.UaaTimeout,
		username: config.Username,
		password: config.Password,
		logger:   config.Logger,
	}

	if err := fetcher.validate(); err != nil {
		return nil, err
	}

	return fetcher, nil
}
