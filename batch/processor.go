package batch

import (
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/retry"
)

// Processor provides a batching mechanism for processing messages efficiently.
// It accumulates individual messages and processes them in batches based
// on size or time thresholds, with support for retries, async processing,
// and response handling.
//
// Usage Example:
//
//	// Create a processor with a handler and configuration
//	processor := batch.NewProcessor(
//	    myHandler,           // Handler that implements ProcessBatch and ProcessOne
//	    responseChan,        // Optional channel to receive processing results
//	    batch.ProcessorConfig{
//	        FlushQueueSize: 100,           // Process when 100 messages accumulate
//	        FlushInterval:  5*time.Second, // Or process every 5 seconds
//	        MaxRetries:     3,             // Retry failed batches up to 3 times
//	        Async:          batch.Async,   // Process batches asynchronously
//	    },
//	)
//
//	// Start the processor (begins listening for messages)
//	processor.Start()
//
//	// Add messages for batch processing
//	processor.Add(message1)
//	processor.Add(message2)
//	// ... messages will be automatically batched and processed
//
//	// Stop the processor (waits for in-flight batches to complete)
//	processor.Stop()
type Processor interface {
	// Start begins the batch processing loop. The processor
	// will start listening for messages and automatically flush batches
	// when FlushQueueSize is reached or FlushInterval elapses.
	// This method is idempotent - calling Start() multiple times
	// has no effect if already running.
	Start()

	// Stop gracefully shuts down the processor. It closes the message channel,
	// waits for all in-flight batches to complete (both sync and async),
	// and prepares for potential restart.
	// This method is idempotent - calling Stop() multiple times
	// has no effect if already stopped.
	Stop()

	// Add queues a message for batch processing. Messages are accumulated
	// until FlushQueueSize is reached or FlushInterval elapses,
	// then processed as a batch by the configured Handler.
	// This method is thread-safe and will block if the internal buffer is full.
	Add(req Message)
}

type processor struct {
	handler  Handler
	reqChan  chan Message
	respChan chan<- Response
	config   ProcessorConfig
	logger   logger.Logger
	retry    retry.Retry
	syncReq  sync.WaitGroup
	asyncReq errgroup.Group
	mu       sync.RWMutex
	running  bool
}

func NewProcessor(
	handler Handler,
	respChan chan<- Response,
	config ProcessorConfig,
) Processor {
	config = applyProcessorConfig(config)

	b := &processor{
		handler:  handler,
		reqChan:  make(chan Message, config.MaxBufferSize),
		respChan: respChan,
		config:   config,
		logger:   config.Logger,
		retry:    config.Retry,
	}
	return b
}

func (p *processor) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return
	}

	p.asyncReq.SetLimit(p.config.MaxAsyncRequests)
	p.asyncReq.Go(func() error {
		p.listen()
		return nil
	})
	p.running = true
}

func (p *processor) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	// initiate exit from the "listen" loop
	close(p.reqChan)

	// wait for all goroutines to finish
	err := p.asyncReq.Wait()
	if err != nil {
		p.logger.Errorf("batch.Processor: failed to wait for all in-flight requests: %v", err)
	}

	// wait for all sync calls to finish
	p.syncReq.Wait()

	// override reqChan to handle a Start->Stop->Start case
	// as next call to Add() will panic if the channel is closed
	p.reqChan = make(chan Message, p.config.MaxBufferSize)
	p.running = false
	p.logger.Debugf("batch.Processor: processed last batch")
}

func (p *processor) Add(req Message) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.reqChan <- req
}

func (p *processor) listen() {
	var batch []Message
	t := time.NewTicker(p.config.FlushInterval)
	defer t.Stop()

	p.logger.Debugf("batch.Processor: listening...")

	var process func(batch []Message)
	if p.config.async {
		process = p.processAsync
	} else {
		process = p.process
	}

	for {
		select {
		case req, ok := <-p.reqChan:
			if !ok {
				if len(batch) > 0 {
					process(batch)
				}
				return
			}
			batch = append(batch, req)
			if len(batch) >= p.config.FlushQueueSize {
				process(batch)
				batch = nil
				t.Reset(p.config.FlushInterval)
			}
		case <-t.C:
			if len(batch) > 0 {
				process(batch)
				batch = nil
			}
		}
	}
}

func (p *processor) processAsync(batch []Message) {
	p.asyncReq.Go(func() error {
		p.process(batch)
		return nil
	})
}

func (p *processor) process(batch []Message) {
	p.syncReq.Add(1)
	defer p.syncReq.Done()

	if len(batch) == 0 {
		return
	}

	var batchReq []Message
	batchReq = append(batchReq, batch...)

	loopErr := p.retry.Do(
		p.config.MaxRetries,
		"batch.Processor.process",
		func(attempt int) (error, retry.ExitStrategy) {
			batchRes, err := p.handler.ProcessBatch(batchReq)
			if err != nil {
				return err, retry.StopNow
			}

			switch status := batchRes.(type) {
			case StatusRetryBatch:
				return status.BatchErr, retry.Continue
			case StatusCannotRetry:
				for _, msgRes := range status.Response {
					p.sendResponse(msgRes)
				}
				batchReq = nil
				return nil, retry.StopNow
			case StatusSuccess:
				for _, msgRes := range status.Response {
					p.sendResponse(msgRes)
				}
				batchReq = nil
				return nil, retry.StopNow
			case StatusPartialSuccess:
				var retryOne []Response
				for _, msgRes := range status.Response {
					if msgRes.Error == nil {
						p.sendResponse(msgRes)
					} else if !msgRes.Retry {
						p.sendResponse(msgRes)
					} else {
						retryOne = append(retryOne, msgRes)
					}
				}
				p.retryIndividual(retryOne)
				batchReq = nil
				return nil, retry.StopNow
			case StatusRetryIndividual:
				var retryOne []Response
				for _, msgRes := range status.Response {
					retryOne = append(retryOne, msgRes)
				}
				p.retryIndividual(retryOne)
				batchReq = nil
				return nil, retry.StopNow
			default:
				p.logger.Errorf("Processor entered an invalid state. Status=%t cannot be handled.", status)
				for _, msgReq := range batchReq {
					p.sendResponse(Response{
						OriginalReq: msgReq,
						Error:       ErrInvalidBatchProcessorState,
						Retry:       false,
					})
				}
				batchReq = nil
				return nil, retry.StopNow
			}
		},
	)

	// Send remaining failed responses.
	// These are the messages that failed all retry attempts.
	for _, msgReq := range batchReq {
		p.sendResponse(Response{
			OriginalReq: msgReq,
			Error:       loopErr,
			Retry:       true,
		})
	}
}

func (p *processor) retryIndividual(retryOne []Response) {
	if len(retryOne) > 0 {
		p.logger.Debugf("Processor will individually retry for %d messages", len(retryOne))
		g := errgroup.Group{}
		g.SetLimit(p.config.NumOfIndividualGoroutines)
		for _, res := range retryOne {
			if p.config.sendIndividual {
				g.Go(func() error {
					resOne := p.handler.ProcessOne(res.OriginalReq)
					p.sendResponse(resOne)
					return nil
				})
			} else {
				p.sendResponse(res)
			}
		}
		waitErr := g.Wait()
		if waitErr != nil {
			p.logger.Errorf(
				"Processor failed to wait for an errgroup (to send messages individually): %v",
				waitErr,
			)
		}
	}
}

func (p *processor) sendResponse(r Response) {
	if p.respChan != nil {
		p.respChan <- r
	}
}
