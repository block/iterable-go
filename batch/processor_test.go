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
				FlushQueueSize:   100,
				FlushInterval:    5 * time.Second,
				MaxRetries:       1,
				Retry:            retry.NewExponentialRetry(),
				sendIndividual:   true,
				MaxBufferSize:    2000,
				async:            true,
				MaxAsyncRequests: 50,
				Logger:           &logger.Noop{},
			},
		},
		{
			name: "override logger",
			inConfig: ProcessorConfig{
				Logger: logger.NewStdOut(),
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:   100,
				FlushInterval:    5 * time.Second,
				MaxRetries:       1,
				Retry:            retry.NewExponentialRetry(),
				sendIndividual:   true,
				MaxBufferSize:    2000,
				async:            true,
				MaxAsyncRequests: 50,
				Logger:           logger.NewStdOut(),
			},
		},
		{
			name: "override retry",
			inConfig: ProcessorConfig{
				Retry: &fakeRetry{},
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:   100,
				FlushInterval:    5 * time.Second,
				MaxRetries:       1,
				Retry:            &fakeRetry{},
				sendIndividual:   true,
				MaxBufferSize:    2000,
				async:            true,
				MaxAsyncRequests: 50,
				Logger:           logger.NewStdOut(),
			},
		},
		{
			name: "override with invalid values",
			inConfig: ProcessorConfig{
				FlushQueueSize:   0,
				FlushInterval:    0 * time.Second,
				MaxRetries:       1,
				MaxBufferSize:    0,
				MaxAsyncRequests: 0,
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:   100,
				FlushInterval:    5 * time.Second,
				MaxRetries:       1,
				Retry:            retry.NewExponentialRetry(),
				sendIndividual:   true,
				MaxBufferSize:    2000,
				async:            true,
				MaxAsyncRequests: 50,
				Logger:           &logger.Noop{},
			},
		},
		{
			name: "override with valid values",
			inConfig: ProcessorConfig{
				FlushQueueSize:   2,
				FlushInterval:    1 * time.Millisecond,
				MaxRetries:       1,
				SendIndividual:   func() bool { return false },
				MaxBufferSize:    1,
				Async:            func() bool { return false },
				MaxAsyncRequests: 2,
			},
			expectConfig: ProcessorConfig{
				FlushQueueSize:   2,
				FlushInterval:    1 * time.Millisecond,
				MaxRetries:       1,
				Retry:            retry.NewExponentialRetry(),
				sendIndividual:   false,
				MaxBufferSize:    1,
				async:            false,
				MaxAsyncRequests: 2,
				Logger:           &logger.Noop{},
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
		msgCnt           int
		expectedBatchCnt int
	}{
		{
			name:             "full batch",
			msgCnt:           testMaxBatchSize,
			expectedBatchCnt: 1,
		},
		{
			name:             "partial batch",
			msgCnt:           testMaxBatchSize - 1,
			expectedBatchCnt: 1,
		},
		{
			name:             "multiple batches",
			msgCnt:           testMaxBatchSize*5 - 1,
			expectedBatchCnt: 5,
		},
		{
			name:             "no batch",
			msgCnt:           0,
			expectedBatchCnt: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &fakeBatchHandler{}
			p, respChan := testBatchProcessor(
				handler,
				testBatchProcessorConfig(IsFalse),
			)

			for i := range tc.msgCnt {
				p.Add(Message{
					Data: strconv.Itoa(i),
				})
			}

			p.Start()
			p.Stop()

			resp := getMessagesOffChan(respChan)

			assert.Equal(t, tc.expectedBatchCnt, handler.batchReqCount)
			assert.Equal(t, tc.msgCnt, handler.totalBatchMessages)
			assert.Equal(t, tc.msgCnt, len(resp))
			assertFailedMessages(t, 0, resp)
		})
	}
}

func TestBatchProcessor_batch_retries(t *testing.T) {
	handler := &fakeBatchHandler{
		shouldFailBatch: failCertainRequests,
	}
	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsFalse),
	)

	// Add a request that will fail
	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: "success",
	})

	p.Start()
	p.Stop()

	resp := getMessagesOffChan(respChan)

	assert.Equal(t, testMaxRetries, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, testMaxRetries*2, handler.totalBatchMessages)
	assert.Equal(t, 2, len(resp))
	assertFailedMessages(t, 2, resp)
}

func TestBatchProcessor_individual_retries(t *testing.T) {
	handler := &fakeBatchHandler{
		shouldFailMsgInBatch: failCertainRequests,
	}
	p, respChan := testBatchProcessor(
		handler,
		testBatchProcessorConfig(IsFalse),
	)

	// Add a request that will fail
	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: "success",
	})

	p.Start()
	p.Stop()

	resp := getMessagesOffChan(respChan)

	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 2, handler.totalBatchMessages)
	assert.Equal(t, 1, handler.nonBatchReqCount)
	assert.Equal(t, 2, len(resp))
	assertFailedMessages(t, 0, resp)
}

func TestBatchProcessor_no_individual_retries(t *testing.T) {
	handler := &fakeBatchHandler{
		shouldFailMsgInBatch: failCertainRequests,
	}
	c := testBatchProcessorConfig(IsFalse)
	c.SendIndividual = func() bool { return false }
	p, respChan := testBatchProcessor(handler, c)

	// Add a request that will fail
	p.Add(Message{
		Data: testFailureMessage,
	})
	p.Add(Message{
		Data: "success",
	})

	p.Start()
	p.Stop()

	resp := getMessagesOffChan(respChan)

	assert.Equal(t, 1, handler.batchReqCount)
	assert.Equal(t, 2, handler.totalBatchMessages)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 2, len(resp))
	assertFailedMessages(t, 1, resp)
}

func TestBatchProcessor_nil_response_chan(t *testing.T) {
	handler := &fakeBatchHandler{}
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
	handler := &fakeBatchHandler{
		shouldFailBatch: failCertainRequests,
	}
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
	resp := getMessagesOffChan(respChan)

	assert.Equal(t, 2_000, handler.batchReqCount)
	assert.Equal(t, 0, handler.nonBatchReqCount)
	assert.Equal(t, 20_000, handler.totalBatchMessages)
	assert.Equal(t, 20_000, len(resp))
	assertFailedMessages(t, 0, resp)

}

func TestBatchProcessor_Start_Stop_Start_Stop(t *testing.T) {
	handler := &fakeBatchHandler{}
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
	handler := &fakeBatchHandler{}
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
	handler := &fakeBatchHandler{}
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
	handler := &fakeBatchHandler{}
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
	batchReqCount        int
	nonBatchReqCount     int
	totalBatchMessages   int
	shouldFailBatch      func(interface{}) bool
	shouldFailMsgInBatch func(interface{}) bool
	shouldFailOne        func(interface{}) bool
	mu                   sync.Mutex
}

var _ Handler = &fakeBatchHandler{}

func (f *fakeBatchHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.batchReqCount++
	f.totalBatchMessages += len(batch)
	res := make([]Response, 0)
	for _, req := range batch {
		if f.shouldFailBatch != nil && f.shouldFailBatch(req.Data) {
			return nil, errors.New("failed"), true
		} else if f.shouldFailMsgInBatch != nil && f.shouldFailMsgInBatch(req.Data) {
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
	return res, nil, false
}

func (f *fakeBatchHandler) ProcessOne(req Message) Response {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nonBatchReqCount++
	if f.shouldFailOne != nil && f.shouldFailOne(req.Data) {
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
