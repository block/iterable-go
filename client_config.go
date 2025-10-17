package iterable_go

import (
	"net/http"
	"time"

	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/rate"
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

	// limiter provides a way to control how many requests are
	// sent to Iterable API
	// default: rate.NoopLimiter
	limiter rate.Limiter
}

func defaultConfig() *config {
	return &config{
		transport: http.DefaultTransport,
		timeout:   10 * time.Second,
		logger:    logger.Noop{},
		limiter:   rate.NoopLimiter{},
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

func WithRateLimiter(limiter rate.Limiter) ConfigOption {
	return func(c *config) {
		c.limiter = limiter
	}
}
