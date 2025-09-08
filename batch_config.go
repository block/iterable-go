package iterable_go

import (
	"time"

	"github.com/block/iterable-go/batch"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/retry"
)

type batchConfig struct {
	// flushQueueSize sets the maximum number of messages
	// to accumulate before triggering a batch flush
	// (maps to ProcessorConfig.FlushQueueSize)
	// default: 100
	flushQueueSize int

	// flushInterval specifies the maximum time to wait
	// before flushing a batch, even if flushQueueSize hasn't been reached
	// (maps to ProcessorConfig.FlushInterval)
	// default: 5 seconds
	flushInterval time.Duration

	// bufferSize determines the buffer size of the internal request channel
	// to prevent blocking on Add() calls
	// (maps to ProcessorConfig.MaxBufferSize)
	// default: 500
	bufferSize int

	// retryTimes sets the maximum number of retry attempts
	// for failed batch operations
	// (maps to ProcessorConfig.MaxRetries)
	// default: 1
	retryTimes int

	// retry configures the retry strategy
	// (exponential backoff, delays, etc.) for failed requests
	// (maps to ProcessorConfig.Retry)
	// default: retry.NewExponentialRetry()
	retry retry.Retry

	// sendAsync determines whether batch processing
	// should run asynchronously or synchronously
	// (maps to ProcessorConfig.Async function)
	// default: true
	sendAsync bool

	// sendIndividual is a function that determines whether the processor
	// needs to send messages one-by-one (individually) if batch request succeeds,
	// but some messages in the batch have errors.
	// This is useful when there's 1 "bad" message in the batch which fails
	// the entire batch.
	// default: true
	sendIndividual bool

	// logger provides logging functionality for debugging
	// and monitoring batch processing operations
	// (maps to ProcessorConfig.Logger)
	// default: logger.Noop
	logger logger.Logger

	// responseChan is an optional channel for receiving
	// batch processing responses and errors
	// (passed to each processor for response handling).
	// If nil - the caller won't get any responses
	// from the batch client.
	// default: nil
	responseChan chan<- batch.Response
}

func defaultBatchConfig() batchConfig {
	return batchConfig{
		flushQueueSize: 100,
		flushInterval:  5 * time.Second,
		bufferSize:     500,
		retryTimes:     1,
		retry:          retry.NewExponentialRetry(),
		sendAsync:      true,
		sendIndividual: true,
		logger:         logger.Noop{},
		responseChan:   nil,
	}
}

type BatchConfigOption func(c *batchConfig)

func WithBatchFlushQueueSize(size int) BatchConfigOption {
	return func(c *batchConfig) {
		c.flushQueueSize = size
	}
}

func WithBatchFlushInterval(interval time.Duration) BatchConfigOption {
	return func(c *batchConfig) {
		c.flushInterval = interval
	}
}

func WithBatchBufferSize(bufferSize int) BatchConfigOption {
	return func(c *batchConfig) {
		c.bufferSize = bufferSize
	}
}

func WithBatchRetryTimes(times int) BatchConfigOption {
	return func(c *batchConfig) {
		c.retryTimes = times
	}
}

func WithBatchRetry(retry retry.Retry) BatchConfigOption {
	return func(c *batchConfig) {
		c.retry = retry
	}
}

func WithBatchSendAsync(sendAsync bool) BatchConfigOption {
	return func(c *batchConfig) {
		c.sendAsync = sendAsync
	}
}

func WithBatchSendIndividual(sendIndividual bool) BatchConfigOption {
	return func(c *batchConfig) {
		c.sendIndividual = sendIndividual
	}
}

func WithBatchLogger(logger logger.Logger) BatchConfigOption {
	return func(c *batchConfig) {
		c.logger = logger
	}
}

func WithBatchResponseListener(res chan batch.Response) BatchConfigOption {
	return func(c *batchConfig) {
		c.responseChan = res
	}
}
