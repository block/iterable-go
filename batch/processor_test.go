package batch

import (
	"cmp"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/retry"
	"github.com/stretchr/testify/assert"
)

const (
	testMaxBatchSize   = 10
	testMaxRetries     = 3
	testFailureMessage = "fail"
)

func TestBatchProcessor_config(t *testing.T) {
	testCases := []struct {
		name         string
		inConfig     ProcessorConfig
		expectConfig ProcessorConfig
	}{
		{
			name:     "default",
			inConfig: ProcessorConfig{},
			expectConfig: ProcessorConfig{
				FlushQueueSize:            100,
				FlushInterval:             5 * time.Second,
				MaxRetries:                1,
				Retry:                     retry.NewExponentialRetry(),
				sendIndividual:            true,
				NumOfIndividualGoroutines: 1,
				MaxBufferSize:             2000,
				async:                     true,
				MaxAsyncRequests:          50,
				Logger:                    &logger.Noop{},
			},
		},
		{
			name: "override logger",
			inConfig: ProcessorConfig{
				Logger: logger.NewStdOut(),
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:            100,
				FlushInterval:             5 * time.Second,
				MaxRetries:                1,
				Retry:                     retry.NewExponentialRetry(),
				sendIndividual:            true,
				NumOfIndividualGoroutines: 1,
				MaxBufferSize:             2000,
				async:                     true,
				MaxAsyncRequests:          50,
				Logger:                    logger.NewStdOut(),
			},
		},
		{
			name: "override retry",
			inConfig: ProcessorConfig{
				Retry: &fakeRetry{},
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:            100,
				FlushInterval:             5 * time.Second,
				MaxRetries:                1,
				Retry:                     &fakeRetry{},
				sendIndividual:            true,
				NumOfIndividualGoroutines: 1,
				MaxBufferSize:             2000,
				async:                     true,
				MaxAsyncRequests:          50,
				Logger:                    logger.NewStdOut(),
			},
		},
		{
			name: "override with invalid values",
			inConfig: ProcessorConfig{
				FlushQueueSize:            0,
				FlushInterval:             0 * time.Second,
				NumOfIndividualGoroutines: 101,
				MaxRetries:                1,
				MaxBufferSize:             0,
				MaxAsyncRequests:          0,
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:            100,
				FlushInterval:             5 * time.Second,
				MaxRetries:                1,
				Retry:                     retry.NewExponentialRetry(),
				sendIndividual:            true,
				NumOfIndividualGoroutines: 1,
				MaxBufferSize:             2000,
				async:                     true,
				MaxAsyncRequests:          50,
				Logger:                    &logger.Noop{},
			},
		},
		{
			name: "override with valid values",
			inConfig: ProcessorConfig{
				FlushQueueSize:            2,
				FlushInterval:             1 * time.Millisecond,
				MaxRetries:                1,
				SendIndividual:            func() bool { return false },
				NumOfIndividualGoroutines: 50,
				MaxBufferSize:             1,
				Async:                     func() bool { return false },
				MaxAsyncRequests:          2,
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:            2,
				FlushInterval:             1 * time.Millisecond,
				MaxRetries:                1,
				Retry:                     retry.NewExponentialRetry(),
				sendIndividual:            false,
				NumOfIndividualGoroutines: 50,
				MaxBufferSize:             1,
				async:                     false,
				MaxAsyncRequests:          2,
				Logger:                    &logger.Noop{},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actual := applyProcessorConfig(tt.inConfig)
			expect := tt.expectConfig

			assert.NotNil(t, actual.Retry)
			assert.NotNil(t, actual.Logger)
			if tt.inConfig.Retry != nil {
				assert.IsType(t, expect.Retry, actual.Retry)
			}
			if tt.inConfig.Logger != nil {
				assert.IsType(t, expect.Logger, actual.Logger)
			}

			// we don't want to compare pointers
			actual.Retry = nil
			actual.Logger = nil
			expect.Retry = nil
			expect.Logger = nil

			// always nil
			assert.Nil(t, actual.SendIndividual)
			assert.Nil(t, actual.Async)

			assert.Equal(t, expect, actual)
		})
	}
}

func TestBatchProcessor_batchSizes(t *testing.T) {
	testCases := []struct {
		name             string
		batchSize        int
		msgCnt           int
		expectedBatchCnt int
	}{
		{
			name:             "full batch",
			batchSize:        10,
			msgCnt:           10,
			expectedBatchCnt: 1,
		},
		{
			name:             "partial batch",
			batchSize:        10,
			msgCnt:           9,
			expectedBatchCnt: 1,
		},
		{
			name:             "multiple batches",
			batchSize:        10,
			msgCnt:           49,
			expectedBatchCnt: 5,
		},
		{
			name:             "no batch",
			batchSize:        10,
			msgCnt:           0,
			expectedBatchCnt: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			handler := newFakeBatchHandler()
			cfg := testBatchProcessorConfig(IsFalse)
			cfg.FlushQueueSize = tt.batchSize

			p, respChan := testBatchProcessor(
				handler, cfg,
			)

			for i := range tt.msgCnt {
				p.Add(Message{
					Data: strconv.Itoa(i),
				})
			}

			p.Start()
			p.Stop()

			res := getMessagesOffChan(respChan)

			assert.Equal(t, tt.expectedBatchCnt, handler.batchReqCount)
			assert.Equal(t, tt.msgCnt, handler.totalBatchMessages)
			assert.Equal(t, tt.msgCnt, len(res))
			assertFailedMessages(t, 0, res)
		})
	}
}

func TestBatchProcessor_batch_fails_no_retry(t *testing.T) {
	handler := newFakeBatchHandler()
	handler.shouldFailBatch = IsTrue

	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsFalse),
	)

	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: testFailureMessage,
	})

	p.Start()
	p.Stop()

	res := getMessagesOffChan(respChan)

	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 2, handler.totalBatchMessages)
	assert.Equal(t, 2, len(res))
	assertFailedMessages(t, 2, res)
}

func TestBatchProcessor_batch_fails_retry(t *testing.T) {
	handler := newFakeBatchHandler()
	handler.shouldRetryBatch = IsTrue

	cfg := testBatchProcessorConfig(IsFalse)
	cfg.MaxRetries = 2

	p, respChan := testBatchProcessor(
		handler,
		cfg,
	)

	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: testFailureMessage,
	})

	p.Start()
	p.Stop()

	res := getMessagesOffChan(respChan)

	assert.Equal(t, 2, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 4, handler.totalBatchMessages)
	assert.Equal(t, 2, len(res))
	assertFailedMessages(t, 2, res)
}

func TestBatchProcessor_batch_fails_retry_one_partial(t *testing.T) {
	handler := newFakeBatchHandler()
	handler.shouldRetryOnePartial = failCertainRequests
	cfg := testBatchProcessorConfig(IsFalse)
	cfg.MaxRetries = 2

	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsFalse),
	)

	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: "success",
	})

	p.Start()
	p.Stop()

	res := getMessagesOffChan(respChan)

	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 2, handler.totalBatchMessages)
	assert.Equal(t, 1, handler.nonBatchReqCount)
	assert.Equal(t, 2, len(res))
	assertFailedMessages(t, 0, res)
}

func TestBatchProcessor_batch_fails_retry_one_all(t *testing.T) {
	handler := newFakeBatchHandler()
	handler.shouldRetryOneAll = failCertainRequests
	cfg := testBatchProcessorConfig(IsFalse)
	cfg.MaxRetries = 2

	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsFalse),
	)

	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: testFailureMessage,
	})

	p.Start()
	p.Stop()

	res := getMessagesOffChan(respChan)

	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 2, handler.totalBatchMessages)
	assert.Equal(t, 2, handler.nonBatchReqCount)
	assert.Equal(t, 2, len(res))
	assertFailedMessages(t, 0, res)
}

func TestBatchProcessor_batch_fails_send_one_is_disabled_1(t *testing.T) {
	handler := newFakeBatchHandler()
	handler.shouldRetryOneAll = failCertainRequests

	c := testBatchProcessorConfig(IsFalse)
	c.SendIndividual = IsFalse
	p, respChan := testBatchProcessor(handler, c)

	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: testFailureMessage,
	})

	p.Start()
	p.Stop()

	res := getMessagesOffChan(respChan)

	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 2, handler.totalBatchMessages)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 2, len(res))
	assertFailedMessages(t, 2, res)
}

func TestBatchProcessor_batch_fails_send_one_is_disabled_2(t *testing.T) {
	handler := newFakeBatchHandler()
	handler.shouldRetryOnePartial = failCertainRequests

	c := testBatchProcessorConfig(IsFalse)
	c.SendIndividual = IsFalse
	p, respChan := testBatchProcessor(handler, c)

	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: "success",
	})

	p.Start()
	p.Stop()

	res := getMessagesOffChan(respChan)

	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 2, handler.totalBatchMessages)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 2, len(res))
	assertFailedMessages(t, 1, res)
}

func TestBatchProcessor_nil_response_chan(t *testing.T) {
	handler := newFakeBatchHandler()
	p := NewProcessor(
		handler,
		nil,
		testBatchProcessorConfig(IsFalse),
	).(*processor)

	p.Start()

	p.Add(Message{
		Data: "success",
	})

	p.Stop()

	assert.Nil(t, p.respChan)
	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 1, handler.totalBatchMessages)
}

func TestBatchProcessor_Start_Stop_Async_race(t *testing.T) {
	respChan := make(chan Response, 100_000)
	handler := newFakeBatchHandler()
	handler.shouldFailBatch = IsTrue

	cfg := testBatchProcessorConfig(IsFalse)
	cfg.FlushQueueSize = 10
	cfg.MaxBufferSize = 100_000
	cfg.MaxRetries = 2
	cfg.Retry = retry.NewExponentialRetry(
		retry.WithInitialDuration(10 * time.Millisecond),
	)
	p := NewProcessor(handler, respChan, cfg).(*processor)

	msgCnt := 20_000
	for i := range msgCnt {
		p.Add(Message{
			Data: strconv.Itoa(i),
		})
	}

	p.Start()
	p.Stop()
	res := getMessagesOffChan(respChan)

	assert.Equal(t, 2_000, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 20_000, handler.totalBatchMessages)
	assert.Equal(t, 20_000, len(res))
	assertFailedMessages(t, 20_000, res)

}

func TestBatchProcessor_Start_Stop_Start_Stop(t *testing.T) {
	handler := newFakeBatchHandler()
	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsTrue),
	)

	p.Add(Message{
		Data: "1",
	})
	p.Start()
	p.Add(Message{
		Data: "2",
	})
	p.Stop()
	p.Add(Message{
		Data: "3",
	})
	p.Start()
	p.Add(Message{
		Data: "4",
	})
	p.Stop()
	p.Add(Message{
		Data: "5",
	})

	res := getMessagesOffChan(respChan)
	assert.Equal(t, 2, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 4, handler.totalBatchMessages)
	assert.Equal(t, 4, len(res))
	assertFailedMessages(t, 0, res)
	assertContainsAll(t, res, []Message{
		{Data: "1"},
		{Data: "2"},
		{Data: "3"},
		{Data: "4"},
		// 5 is missing, because it was sent after the last Stop()
	})
}

func TestBatchProcessor_Stop_Stop_Start_Stop(t *testing.T) {
	handler := newFakeBatchHandler()
	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsTrue),
	)

	p.Add(Message{
		Data: "1",
	})
	p.Stop()
	p.Add(Message{
		Data: "2",
	})
	p.Stop()
	p.Add(Message{
		Data: "3",
	})
	p.Start()
	p.Add(Message{
		Data: "4",
	})
	p.Stop()
	p.Add(Message{
		Data: "5",
	})

	res := getMessagesOffChan(respChan)
	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 4, handler.totalBatchMessages)
	assert.Equal(t, 4, len(res))
	assertFailedMessages(t, 0, res)
	assertContainsAll(t, res, []Message{
		{Data: "1"},
		{Data: "2"},
		{Data: "3"},
		{Data: "4"},
		// 5 is missing, because it was sent after the last Stop()
	})
}

func TestBatchProcessor_Start_Start_Stop(t *testing.T) {
	handler := newFakeBatchHandler()
	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsTrue),
	)

	p.Add(Message{
		Data: "1",
	})
	p.Start()
	p.Add(Message{
		Data: "2",
	})
	p.Start()
	p.Add(Message{
		Data: "3",
	})
	p.Start()
	p.Add(Message{
		Data: "4",
	})
	p.Stop()
	p.Add(Message{
		Data: "5",
	})

	res := getMessagesOffChan(respChan)
	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 4, handler.totalBatchMessages)
	assert.Equal(t, 4, len(res))
	assertFailedMessages(t, 0, res)

	assertContainsAll(t, res, []Message{
		{Data: "1"},
		{Data: "2"},
		{Data: "3"},
		{Data: "4"},
		// 5 is missing, because it was sent after the last Stop()
	})
}

func TestBatchProcessor_Stop_Add_race(t *testing.T) {
	respChan := make(chan Response, 100_000)
	handler := newFakeBatchHandler()
	cfg := testBatchProcessorConfig(IsFalse)
	cfg.FlushQueueSize = 2
	cfg.MaxBufferSize = 100_000
	p := NewProcessor(handler, respChan, cfg).(*processor)
	p.Start()

	wg := sync.WaitGroup{}
	msgCnt := 1000
	for i := range msgCnt {
		wg.Add(1)
		go func() {
			p.Add(Message{
				Data: strconv.Itoa(i),
			})
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			p.Stop()
			p.Start()
		}()
	}

	// last Stop() guarantees all messages are delivered
	p.Stop()
	resp := getMessagesOffChan(respChan)

	assert.Equal(t, 1000, handler.totalBatchMessages)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 1000, len(resp))
	assertFailedMessages(t, 0, resp)
}

func assertFailedMessages(t *testing.T, expectFailed int, res []Response) {
	actualFailed := 0
	for _, r := range res {
		if r.Error != nil {
			actualFailed++
		}
	}
	assert.Equal(t, expectFailed, actualFailed)
}

func assertContainsAll(t *testing.T, res []Response, messages []Message) bool {
	assert.Equal(t, len(res), len(messages))
	slices.SortFunc(res, func(a, b Response) int {
		return cmp.Compare(
			a.OriginalReq.Data.(string),
			b.OriginalReq.Data.(string),
		)
	})
	slices.SortFunc(messages, func(a, b Message) int {
		return cmp.Compare(
			a.Data.(string),
			b.Data.(string),
		)
	})
	for i, r := range res {
		assert.Equal(t, r.OriginalReq, messages[i])
	}
	return true
}

func getMessagesOffChan(c chan Response) []Response {
	// close the channel to make sure the loop below exits
	close(c)

	msgs := make([]Response, 0)
	for msg := range c {
		msgs = append(msgs, msg)
	}
	return msgs
}

type fakeBatchHandler struct {
	batchReqCount         int
	nonBatchReqCount      int
	totalBatchMessages    int
	shouldFailBatch       func() bool
	shouldRetryBatch      func() bool
	shouldRetryOnePartial func(any) bool
	shouldRetryOneAll     func(any) bool
	shouldFailOne         func() bool
	mu                    sync.Mutex
}

var _ Handler = &fakeBatchHandler{}

func newFakeBatchHandler() *fakeBatchHandler {
	return &fakeBatchHandler{
		batchReqCount:      0,
		nonBatchReqCount:   0,
		totalBatchMessages: 0,
		shouldFailBatch: func() bool {
			return false
		},
		shouldRetryBatch: func() bool {
			return false
		},
		shouldRetryOnePartial: func(_ any) bool {
			return false
		},
		shouldRetryOneAll: func(_ any) bool {
			return false
		},
		shouldFailOne: func() bool {
			return false
		},
	}
}

func (f *fakeBatchHandler) ProcessBatch(batch []Message) (ProcessBatchResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.batchReqCount++
	f.totalBatchMessages += len(batch)
	res := make([]Response, 0)
	failBatch := false
	retryBatch := false
	retryOnePartial := false
	retryOneAll := false
	for _, req := range batch {
		if f.shouldFailBatch() {
			failBatch = true
			res = append(res, Response{
				Error:       errors.New("failed"),
				OriginalReq: req,
				Retry:       true,
			})
		} else if f.shouldRetryBatch() {
			retryBatch = true
			res = append(res, Response{
				Error:       errors.New("failed"),
				OriginalReq: req,
				Retry:       true,
			})
		} else if f.shouldRetryOnePartial(req.Data) {
			retryOnePartial = true
			res = append(res, Response{
				Error:       errors.New("failed"),
				OriginalReq: req,
				Retry:       true,
			})
		} else if f.shouldRetryOneAll(req.Data) {
			retryOneAll = true
			res = append(res, Response{
				Error:       errors.New("failed"),
				OriginalReq: req,
				Retry:       true,
			})
		} else {
			res = append(res, Response{
				OriginalReq: req,
			})
		}
	}

	if failBatch {
		return StatusCannotRetry{res}, nil
	}
	if retryBatch {
		return StatusRetryBatch{res, errors.New("failed")}, nil
	}
	if retryOnePartial {
		return StatusPartialSuccess{res}, nil
	}
	if retryOneAll {
		return StatusRetryIndividual{res}, nil
	}

	return StatusSuccess{res}, nil
}

func (f *fakeBatchHandler) ProcessOne(req Message) Response {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nonBatchReqCount++
	if f.shouldFailOne() {
		return Response{
			Error:       errors.New("failed"),
			OriginalReq: req,
		}
	}
	return Response{OriginalReq: req}
}

func failCertainRequests(data interface{}) bool {
	shouldFail := true
	if str, ok := data.(string); ok {
		shouldFail = str == testFailureMessage
	}
	return shouldFail
}

func testBatchProcessor(
	handler Handler,
	config ProcessorConfig,
) (*processor, chan Response) {
	respChan := make(chan Response, testMaxBatchSize*10)
	return NewProcessor(
		handler,
		respChan,
		config,
	).(*processor), respChan
}

func testBatchProcessorConfig(isAsync BoolFunc) ProcessorConfig {
	return ProcessorConfig{
		FlushQueueSize: testMaxBatchSize,
		FlushInterval:  500 * time.Millisecond,
		MaxRetries:     testMaxRetries,
		Retry: retry.NewExponentialRetry(
			retry.WithInitialDuration(1 * time.Millisecond),
		),
		MaxBufferSize: testMaxBatchSize * 10,
		Async:         isAsync,
		Logger:        &logger.Noop{},
	}
}

type fakeRetry struct{}

var _ retry.Retry = &fakeRetry{}

func (f fakeRetry) Do(_ int, _ string, _ retry.RetriableFn) error {
	return nil
}
