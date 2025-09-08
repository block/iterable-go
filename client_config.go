package iterable_go

import (
	"net/http"
	"time"

	"github.com/block/iterable-go/logger"
)

type config struct {
	// transport specifies the HTTP transport mechanism
	// for making requests.
	// It's useful for mocking or if customers
	// want to add extra logging, headers, etc.
	// default: http.DefaultTransport
	transport http.RoundTripper

	// timeout sets the maximum duration for HTTP requests
	// before they are cancelled
	// default: 10 seconds
	timeout time.Duration

	// logger provides logging functionality for all internal
	// iterable-go client operations
	// default: logger.Noop
	logger logger.Logger
}

func defaultConfig() *config {
	return &config{
		transport: http.DefaultTransport,
		timeout:   10 * time.Second,
		logger:    logger.Noop{},
	}
}

type ConfigOption func(c *config)

func WithTransport(transport http.RoundTripper) ConfigOption {
	return func(c *config) {
		c.transport = transport
	}
}

func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *config) {
		c.timeout = timeout
	}
}

func WithLogger(logger logger.Logger) ConfigOption {
	return func(c *config) {
		c.logger = logger
	}
}
