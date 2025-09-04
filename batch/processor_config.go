package batch

import (
	"time"

	"iterable-go/logger"
	"iterable-go/retry"
)

type ProcessorConfig struct {
	// FlushQueueSize defines the maximum number of messages
	// to accumulate before triggering a batch flush
	// default: 100
	FlushQueueSize int

	// FlushInterval specifies the maximum time to wait
	// before flushing a batch, even if FlushQueueSize hasn't been reached
	// default: 5 seconds
	FlushInterval time.Duration

	// MaxRetries sets the maximum number of retry attempts
	// for failed batch operations
	// default: 1
	MaxRetries int

	// Retry configures the retry strategy (exponential backoff, delays, etc.)
	// for failed requests
	// default: retry.NewExponentialRetry
	Retry retry.Retry

	// sendIndividual is an internal field that stores the resolved mode
	// from the SendIndividual function
	// default: true
	sendIndividual bool

	// SendIndividual is a function that determines whether the processor
	// needs to send messages one-by-one (individually) if batch request succeeds,
	// but some messages in the batch have errors.
	// This is useful when there's 1 "bad" message in the batch which fails
	// the entire batch.
	// default: true
	SendIndividual BoolFunc

	// MaxBufferSize determines the buffer size of the internal request channel
	// to prevent blocking on Add() calls
	// default: 2000
	MaxBufferSize int

	// async is an internal field that stores the resolved async mode
	// from the Async function
	// default: true
	async bool

	// Async is a function that determines whether batch processing
	// should run asynchronously or synchronously
	// default: true
	Async BoolFunc

	// MaxAsyncRequests limits the number of concurrent goroutines
	// when processing batches asynchronously.
	// default: 50
	MaxAsyncRequests int

	// Logger provides logging functionality for debugging
	// and monitoring batch processing operations
	// default: logger.Noop
	Logger logger.Logger
}

type BoolFunc func() bool

var IsFalse BoolFunc = func() bool { return false }
var IsTrue BoolFunc = func() bool { return true }

func defaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		FlushQueueSize: 100,
		FlushInterval:  5 * time.Second,
		MaxRetries:     1,
		Retry: retry.NewExponentialRetry(
			retry.WithInitialDuration(100*time.Millisecond),
			retry.WithLogger(&logger.Noop{}),
		),
		sendIndividual:   true,
		MaxBufferSize:    2000,
		async:            true,
		MaxAsyncRequests: 50,
		Logger:           &logger.Noop{},
	}
}

func applyProcessorConfig(inConfig ProcessorConfig) ProcessorConfig {
	outConfig := defaultProcessorConfig()
	if inConfig.FlushQueueSize > 0 {
		outConfig.FlushQueueSize = inConfig.FlushQueueSize
	}
	if inConfig.FlushInterval > 0 {
		outConfig.FlushInterval = inConfig.FlushInterval
	}
	if inConfig.MaxRetries > 0 {
		outConfig.MaxRetries = inConfig.MaxRetries
	}
	if inConfig.Retry != nil {
		outConfig.Retry = inConfig.Retry
	}
	if inConfig.SendIndividual != nil {
		outConfig.sendIndividual = inConfig.SendIndividual()
	}
	if inConfig.MaxBufferSize > 0 {
		outConfig.MaxBufferSize = inConfig.MaxBufferSize
	}
	if inConfig.Async != nil {
		outConfig.async = inConfig.Async()
	}
	if inConfig.MaxAsyncRequests > 1 {
		// processor needs at least 2 goroutines:
		// 1 - for the listener itself
		// 2 - for the async request
		outConfig.MaxAsyncRequests = inConfig.MaxAsyncRequests
	}
	if inConfig.Logger != nil {
		outConfig.Logger = inConfig.Logger
	}

	return outConfig
}
