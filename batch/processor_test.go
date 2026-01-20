package batch

import (
	"cmp"
	"errors"
	"fmt"
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

func TestBatchProcessor_errors_Join_and_Is_work(t *testing.T) {
	err1 := errors.Join(ErrBatchError, fmt.Errorf("boo"))
	err2 := errors.Join(errors.Join(ErrBatchError), fmt.Errorf("boo"))
	err3 := errors.Join(errors.Join(ErrBatchError, nil), fmt.Errorf("boo"))

	assert.ErrorIs(t, err1, ErrBatchError)
	assert.True(t, errors.Is(err1, ErrBatchError))
	assert.ErrorIs(t, err2, ErrBatchError)
	assert.True(t, errors.Is(err2, ErrBatchError))
	assert.ErrorIs(t, err3, ErrBatchError)
	assert.True(t, errors.Is(err3, ErrBatchError))
}

func TestBatchProcessor_errors(t *testing.T) {
	testCases := []struct {
		name                string
		sendIndividual      bool
		messages            []Message
		batchResponseQ      []fakeBatchResponse
		oneResponseQ        []Response
		expectBatchReqCount int
		expectOneReqCount   int
		expectMsg           []string
		expectMsgRetry      []string
		expectMsgErr        []string
		expectBatchErr      []string

		expectMsgRetryCount int
		expectMsgErrorCount int
		expectBatchErrType  int
	}{
		{
			name: "StatusSuccess",
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusSuccess{
						Response: []Response{
							{Data: "1", OriginalReq: Message{Data: "1"}},
							{Data: "2", OriginalReq: Message{Data: "2"}},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
		},
		{
			name: "batch error",
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: nil,
					err: errors.New("generic batch error"),
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{"1", "2"},
			expectMsgErr:        []string{"1", "2"},
			expectBatchErr:      []string{"1", "2"},
		},
		{
			name: "StatusRetryBatch, StatusSuccess",
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryBatch{
						BatchErr: errors.New("fail"),
						Response: []Response{
							{Data: "1", OriginalReq: Message{Data: "1"}},
							{Data: "2", OriginalReq: Message{Data: "2"}},
						},
					},
					err: nil,
				},
				{
					res: StatusSuccess{
						Response: []Response{
							{Data: "1", OriginalReq: Message{Data: "1"}},
							{Data: "2", OriginalReq: Message{Data: "2"}},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 2,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "StatusSuccess - with errors and retries",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusSuccess{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Error:       errors.New("fail 1"), // will be set to nil
								Retry:       false,
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
								Error:       errors.New("fail 2"), // will be set to nil
								Retry:       true,                 // will be set to false
							},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name: "StatusRetryBatch, StatusRetryBatch",
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryBatch{
						BatchErr: errors.New("fail 1"),
						Response: []Response{
							{Data: "1", OriginalReq: Message{Data: "1"}},
							{Data: "2", OriginalReq: Message{Data: "2"}},
						},
					},
					err: nil,
				},
				{
					res: StatusRetryBatch{
						BatchErr: errors.New("fail 2"),
						Response: []Response{
							{Data: "1", OriginalReq: Message{Data: "1"}},
							{Data: "2", OriginalReq: Message{Data: "2"}},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 2,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{"1", "2"},
			expectMsgErr:        []string{"1", "2"},
			expectBatchErr:      []string{"1", "2"},
		},
		{
			name: "StatusCannotRetry",
			messages: []Message{
				{Data: "1"}, {Data: "2"}, {Data: "3"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusCannotRetry{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Retry:       true, // this should be set to false
								Error:       errors.New("fail 1"),
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
							},
							{
								Data:        "3",
								OriginalReq: Message{Data: "3"},
								Error:       errors.New("fail 3"),
							},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2", "3"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{"1", "2", "3"},
			expectBatchErr:      []string{"1", "2", "3"},
		},
		{
			name: "StatusRetryBatch, StatusCannotRetry",
			messages: []Message{
				{Data: "1"}, {Data: "2"}, {Data: "3"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryBatch{
						BatchErr: errors.New("fail batch 1"),
						Response: []Response{
							{Data: "1", OriginalReq: Message{Data: "1"}},
							{Data: "2", OriginalReq: Message{Data: "2"}},
							{Data: "3", OriginalReq: Message{Data: "3"}},
						},
					},
					err: nil,
				},
				{
					res: StatusCannotRetry{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Retry:       true, // this should be set to false
								Error:       errors.New("fail 1"),
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
							},
							{
								Data:        "3",
								OriginalReq: Message{Data: "3"},
								Error:       errors.New("fail 3"),
							},
						},
					},
					err: nil,
				},
				{
					res: StatusRetryBatch{
						BatchErr: errors.New("fail 2"),
						Response: []Response{
							{Data: "1", OriginalReq: Message{Data: "1"}},
							{Data: "2", OriginalReq: Message{Data: "2"}},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 2,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2", "3"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{"1", "2", "3"},
			expectBatchErr:      []string{"1", "2", "3"},
		},
		{
			name:           "StatusPartialSuccess, sendIndividual:false",
			sendIndividual: false,
			messages: []Message{
				{Data: "1"}, {Data: "2"}, {Data: "3"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusPartialSuccess{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Error:       errors.New("fail 1"),
								Retry:       false,
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
								Error:       errors.New("fail 2"),
								Retry:       true,
							},
							{
								Data:        "3",
								OriginalReq: Message{Data: "3"},
							},
						},
					},
					err: nil,
				},
			},
			oneResponseQ: []Response{
				{
					Data:        "1",
					OriginalReq: Message{Data: "1"},
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2", "3"},
			expectMsgRetry:      []string{"2"},
			expectMsgErr:        []string{"1", "2"},
			expectBatchErr:      []string{"1", "2"},
		},
		{
			name:           "StatusPartialSuccess, sendIndividual:true",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"}, {Data: "3"}, {Data: "4"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusPartialSuccess{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Error:       errors.New("fail 1"),
								Retry:       false,
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
								Error:       errors.New("fail 2"),
								Retry:       true,
							},
							{
								Data:        "3",
								OriginalReq: Message{Data: "3"},
								Retry:       true, // will be set to false, because Error is nil
							},
							{
								Data:        "4",
								OriginalReq: Message{Data: "4"},
							},
						},
					},
					err: nil,
				},
			},
			oneResponseQ: []Response{
				{
					Data:        "2",
					OriginalReq: Message{Data: "2"},
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   1,
			expectMsg:           []string{"1", "2", "3", "4"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{"1"},
			expectBatchErr:      []string{"1"},
		},
		{
			name:           "StatusPartialSuccess - no errors",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusPartialSuccess{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
							},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "StatusPartialSuccess - all errors, sendIndividual:true",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusPartialSuccess{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Error:       errors.New("fail 1"),
								Retry:       true,
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
								Error:       errors.New("fail 2"),
								Retry:       true,
							},
						},
					},
					err: nil,
				},
			},
			oneResponseQ: []Response{
				{
					Data:        "1",
					OriginalReq: Message{Data: "1"},
				},
				{
					Data:        "2",
					OriginalReq: Message{Data: "2"},
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   2,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "StatusPartialSuccess - empty",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusPartialSuccess{
						Response: []Response{
							// this should never happen, but if it does,
							// let's make sure Processor doesn't panic
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "StatusRetryIndividual, sendIndividual:false",
			sendIndividual: false,
			messages: []Message{
				{Data: "1"}, {Data: "2"}, {Data: "3"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryIndividual{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Error:       errors.New("fail 1"),
								Retry:       false,
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
								Error:       errors.New("fail 2"),
								Retry:       true,
							},
							{
								Data:        "3",
								OriginalReq: Message{Data: "3"},
							},
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2", "3"},
			expectMsgRetry:      []string{"1", "2", "3"},
			expectMsgErr:        []string{"1", "2", "3"},
			expectBatchErr:      []string{"1", "2", "3"},
		},
		{
			name:           "StatusRetryIndividual, sendIndividual:true",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"}, {Data: "3"}, {Data: "4"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryIndividual{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Error:       errors.New("fail 1"),
								Retry:       false,
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
								Error:       errors.New("fail 2"),
								Retry:       true,
							},
							{
								Data:        "3",
								OriginalReq: Message{Data: "3"},
								Retry:       true, // will be set to false, because Error is nil
							},
							{
								Data:        "4",
								OriginalReq: Message{Data: "4"},
							},
						},
					},
					err: nil,
				},
			},
			oneResponseQ: []Response{
				{
					Data:        "1",
					OriginalReq: Message{Data: "1"},
				},
				{
					Data:        "2",
					OriginalReq: Message{Data: "2"},
				},
				{
					Data:        "3",
					OriginalReq: Message{Data: "3"},
				},
				{
					Data:        "4",
					OriginalReq: Message{Data: "4"},
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   4,
			expectMsg:           []string{"1", "2", "3", "4"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "StatusRetryIndividual - no errors, sendIndividual: true",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryIndividual{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
							},
						},
					},
					err: nil,
				},
			},
			oneResponseQ: []Response{
				{
					Data:        "1",
					OriginalReq: Message{Data: "1"},
				},
				{
					Data:        "2",
					OriginalReq: Message{Data: "2"},
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   2,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "StatusRetryIndividual - all errors, sendIndividual:true",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryIndividual{
						Response: []Response{
							{
								Data:        "1",
								OriginalReq: Message{Data: "1"},
								Error:       errors.New("fail 1"),
								Retry:       true,
							},
							{
								Data:        "2",
								OriginalReq: Message{Data: "2"},
								Error:       errors.New("fail 2"),
								Retry:       false,
							},
						},
					},
					err: nil,
				},
			},
			oneResponseQ: []Response{
				{
					Data:        "1",
					OriginalReq: Message{Data: "1"},
				},
				{
					Data:        "2",
					OriginalReq: Message{Data: "2"},
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   2,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "StatusRetryIndividual - empty",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{
					res: StatusRetryIndividual{
						Response: []Response{
							// this should never happen, but if it does,
							// let's make sure Processor doesn't panic
						},
					},
					err: nil,
				},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{},
			expectBatchErr:      []string{},
		},
		{
			name:           "InvalidBatchResponse",
			sendIndividual: true,
			messages: []Message{
				{Data: "1"}, {Data: "2"},
			},
			batchResponseQ: []fakeBatchResponse{
				{res: invalidBatchResponse{}},
			},
			expectBatchReqCount: 1,
			expectOneReqCount:   0,
			expectMsg:           []string{"1", "2"},
			expectMsgRetry:      []string{},
			expectMsgErr:        []string{"1", "2"},
			expectBatchErr:      []string{"1", "2"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			handler := newFakeBatchHandler()
			handler.processBatchQ = tt.batchResponseQ

			cfg := testBatchProcessorConfig(IsFalse)
			cfg.MaxBufferSize = 10
			cfg.MaxRetries = 2
			if tt.sendIndividual {
				cfg.SendIndividual = IsTrue
			} else {
				cfg.SendIndividual = IsFalse
			}

			p, resChan := testBatchProcessor(
				handler, cfg,
			)

			for _, m := range tt.messages {
				p.Add(m)
			}

			p.Start()
			p.Stop()

			res := getMessagesOffChan(resChan)

			assert.NotNil(t, res)
			assert.Equal(t, tt.expectBatchReqCount, handler.batchReqCount)
			assert.Equal(t, tt.expectOneReqCount, handler.nonBatchReqCount)

			assert.Equal(t, len(tt.expectMsg), len(res))
			for _, r := range res {
				data := r.OriginalReq.Data.(string)
				assert.Contains(t, tt.expectMsg, data)
				idx := slices.Index(tt.expectMsg, data)
				if idx > -1 {
					tt.expectMsg = slices.Delete(tt.expectMsg, idx, idx+1)
				}

				if r.Retry {
					assert.Contains(t, tt.expectMsgRetry, data)
					idx = slices.Index(tt.expectMsgRetry, data)
					if idx > -1 {
						tt.expectMsgRetry = slices.Delete(tt.expectMsgRetry, idx, idx+1)
					}
				}

				if r.Error != nil {
					assert.Contains(t, tt.expectMsgErr, data)
					idx = slices.Index(tt.expectMsgErr, data)
					if idx > -1 {
						tt.expectMsgErr = slices.Delete(tt.expectMsgErr, idx, idx+1)
					}
				}

				if errors.Is(r.Error, ErrBatchError) {
					assert.Contains(t, tt.expectBatchErr, data)
					idx = slices.Index(tt.expectBatchErr, data)
					if idx > -1 {
						tt.expectBatchErr = slices.Delete(tt.expectBatchErr, idx, idx+1)
					}
				}
			}
			assert.Empty(t, tt.expectMsg)
			assert.Empty(t, tt.expectMsgRetry)
			assert.Empty(t, tt.expectMsgErr)
			assert.Empty(t, tt.expectBatchErr)
		})
	}
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

type fakeBatchResponse struct {
	res ProcessBatchResponse
	err error
}

type fakeBatchHandler struct {
	batchReqCount      int
	nonBatchReqCount   int
	totalBatchMessages int
	processBatchQ      []fakeBatchResponse
	processOneQ        []Response

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

	if len(f.processBatchQ) > 0 {
		batchRes := f.processBatchQ[0]
		f.processBatchQ = f.processBatchQ[1:]
		f.processBatchQ = append(
			f.processBatchQ,
			// the test should never reach this point
			// if it does - we must return an error to make sure
			// the 2 ways of handling responses don't affect each other's results
			fakeBatchResponse{
				err: errors.New("cannot mix processBatchQ with 'shouldFail'"),
			},
		)
		return batchRes.res, batchRes.err
	}

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

	var res Response
	if len(f.processOneQ) > 0 {
		res = f.processOneQ[0]
		f.processOneQ = f.processOneQ[1:]
		// the test should never reach this point
		// if it does - we must return an error to make sure
		// the 2 ways of handling responses don't affect each other's results
		f.processOneQ = append(
			f.processOneQ, Response{
				Data: "cannot mix processBatchQ with 'shouldFail'",
				OriginalReq: Message{
					Data: "cannot mix processBatchQ with 'shouldFail'",
				},
				Error: errors.New("cannot mix processBatchQ with 'shouldFail'"),
				Retry: true,
			},
		)
		return res
	}

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

type invalidBatchResponse struct{}

func (r invalidBatchResponse) response() []Response {
	return nil
}

var _ ProcessBatchResponse = &invalidBatchResponse{}
