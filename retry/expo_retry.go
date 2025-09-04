package retry

import (
	"fmt"
	"time"

	"iterable-go/logger"
)

type expoConfig struct {
	sleep  time.Duration
	logger logger.Logger
}

func defaultExpoConfig() expoConfig {
	return expoConfig{
		sleep:  50 * time.Millisecond,
		logger: &logger.Noop{},
	}
}

type ExpoConfigOption func(c *expoConfig)

func WithLogger(log logger.Logger) ExpoConfigOption {
	return func(c *expoConfig) {
		c.logger = log
	}
}

func WithInitialDuration(d time.Duration) ExpoConfigOption {
	return func(c *expoConfig) {
		c.sleep = d
	}
}

type expoRetry struct {
	config expoConfig
}

var _ Retry = &expoRetry{}

func NewExponentialRetry(opts ...ExpoConfigOption) Retry {
	var config = defaultExpoConfig()
	for _, opt := range opts {
		opt(&config)
	}

	return &expoRetry{config}
}

// Do runs provided function repeatedly until:
// * the RetriableFn returns no error
// * or attempts is reached
// * or RetriableFn returns StopNow
// Examples:
// Do(3, "my-func", func(attempt int) (error, retry.ExitStrategy) {})
// ^ will run the function 3 times, sleeping 0ms, 50ms, 100ms before each run.
//
// Do(0, "my-func", func(attempt int) (error, retry.ExitStrategy) {})
// ^ will NOT run
func (r *expoRetry) Do(
	attempts int,
	fnName string,
	fn RetriableFn,
) error {
	if attempts < 1 {
		return fmt.Errorf("attempts must be > 0")
	}

	var err error
	var i int

	sleep := r.config.sleep
	for i < attempts {
		var exitNow ExitStrategy
		if err, exitNow = fn(i); err == nil {
			return nil
		}
		if exitNow {
			return err
		}

		r.config.logger.Warnf(
			"Error during retry %s; retrying. attempt=%d, maxAttempt=%d, backoff=%v, error=%v",
			fnName, i, attempts, sleep, err,
		)

		time.Sleep(sleep)

		sleep = sleep * 2
		i++
	}

	r.config.logger.Warnf(
		"Exhausted all retry attempts for %s; giving up. attempt=%d, maxAttempt=%d, backoff=%v, error=%v",
		fnName, i, attempts, sleep, err,
	)

	return err
}
