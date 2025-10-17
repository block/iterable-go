package batch

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/block/iterable-go/api"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/rate"
	"github.com/block/iterable-go/types"

	"github.com/stretchr/testify/assert"
)

func TestListUnSubscribeBatchHandler_ProcessBatch(t *testing.T) {
	tests := []struct {
		name                    string
		failCnt                 int
		rateLimitCnt            int
		uniqueIds               bool
		noEmailCnt              int
		expectedErr             bool
		expectedRetry           bool
		expectedReqCnt          int
		expectedResLen          int
		expectedIndividualRetry int
	}{
		{
			name:                    "Success",
			failCnt:                 0,
			rateLimitCnt:            0,
			uniqueIds:               false,
			expectedErr:             false,
			expectedRetry:           false,
			expectedReqCnt:          1,
			expectedResLen:          10,
			expectedIndividualRetry: 0,
		},
		{
			name:                    "UniqueIds",
			failCnt:                 0,
			rateLimitCnt:            0,
			uniqueIds:               true,
			expectedErr:             false,
			expectedRetry:           false,
			expectedReqCnt:          10,
			expectedResLen:          10,
			expectedIndividualRetry: 0,
		},
		{
			name:                    "Fail",
			failCnt:                 2,
			rateLimitCnt:            0,
			uniqueIds:               false,
			expectedErr:             false,
			expectedRetry:           false,
			expectedReqCnt:          3,
			expectedResLen:          10,
			expectedIndividualRetry: 0,
		},
		{
			name:                    "RateLimit",
			failCnt:                 0,
			rateLimitCnt:            3,
			uniqueIds:               false,
			expectedErr:             false,
			expectedRetry:           false,
			expectedReqCnt:          3,
			expectedResLen:          10,
			expectedIndividualRetry: 10,
		},
		{
			name:                    "FailUnique",
			failCnt:                 3,
			rateLimitCnt:            0,
			uniqueIds:               true,
			expectedErr:             false,
			expectedRetry:           false,
			expectedReqCnt:          12,
			expectedResLen:          10,
			expectedIndividualRetry: 1,
			noEmailCnt:              1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, transport := testListUnSubscribeHandler(tt.failCnt, tt.rateLimitCnt)
			batch := generateListUnSubTestBatchMessages(10, tt.uniqueIds, tt.noEmailCnt)
			res, err, retry := handler.ProcessBatch(batch)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResLen, len(res))
			assert.Equal(t, tt.expectedRetry, retry)
			assert.Equal(t, tt.expectedReqCnt, transport.reqCnt)
			retryCount := 0
			for _, r := range res {
				if r.Retry {
					retryCount++
				}
			}
			assert.Equal(t, tt.expectedIndividualRetry, retryCount)
		})
	}
}

func TestListUnSubscribeBatchHandler_ProcessBatch_DuplicateEmail(t *testing.T) {
	handler, transport := testListUnSubscribeHandler(0, 0)
	batch := []Message{
		{
			Data: &types.ListUnSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListUnSubscriber{
					{
						Email: testEmail,
					},
				},
			},
		},
		{
			Data: &types.ListUnSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListUnSubscriber{
					{
						Email: testEmail,
					},
				},
			},
		},
	}
	res, err, retry := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, false, retry)

	successCnt := 0
	failCnt := 0
	for _, r := range res {
		if r.Error == nil {
			successCnt++
		} else {
			failCnt++
		}
	}
	assert.Equal(t, 2, successCnt)
	assert.Equal(t, 0, failCnt)
	assert.Equal(t, transport.reqCnt, 1)
}

func TestListUnSubscribeBatchHandler_ProcessBatch_DuplicateId(t *testing.T) {
	handler, transport := testListUnSubscribeHandler(0, 0)
	batch := []Message{
		{
			Data: &types.ListUnSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListUnSubscriber{
					{
						UserId: testUserId,
					},
				},
			},
		},
		{
			Data: &types.ListUnSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListUnSubscriber{
					{
						UserId: testUserId,
					},
				},
			},
		},
	}
	res, err, retry := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, false, retry)

	successCnt := 0
	failCnt := 0
	for _, r := range res {
		if r.Error == nil {
			successCnt++
		} else {
			failCnt++
		}
	}
	assert.Equal(t, 2, successCnt)
	assert.Equal(t, 0, failCnt)
	assert.Equal(t, transport.reqCnt, 1)
}

func TestListUnSubscribeBatchHandler_ProcessBatch_InvalidData(t *testing.T) {
	handler, transport := testListUnSubscribeHandler(0, 0)
	batch := []Message{
		{
			Data: "invalid",
		},
	}
	res, err, retry := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, false, retry)
	assert.Equal(t, transport.reqCnt, 0)

	assert.Error(t, res[0].Error)
}

func TestListUnSubscribeBatchHandler_ProcessOne(t *testing.T) {
	handler, transport := testListUnSubscribeHandler(0, 0)
	batch := generateListUnSubTestBatchMessages(10, false, 0)

	res := make([]Response, 0, len(batch))
	for _, req := range batch {
		newRes := handler.ProcessOne(req)
		assert.True(t, newRes.Retry)
		res = append(res, newRes)
	}

	assert.Equal(t, 10, len(res))
	assert.Equal(t, transport.reqCnt, 0)
}

func TestListUnSubscribeBatchHandler_ProcessOne_InvalidData(t *testing.T) {
	handler, transport := testListUnSubscribeHandler(0, 0)
	req := Message{
		Data: "invalid",
	}
	res := handler.ProcessOne(req)

	assert.Equal(t, transport.reqCnt, 0)
	assert.Error(t, res.Error)
}

func generateListUnSubTestBatchMessages(cnt int, uniqueIds bool, noEmailCnt int) []Message {
	var batch []Message
	listId := testListId
	for i := range cnt {
		if uniqueIds {
			listId = i
		}
		if i < noEmailCnt {
			batch = append(batch, Message{
				Data: &types.ListUnSubscribeRequest{
					ListId: int64(listId),
					Subscribers: []types.ListUnSubscriber{
						{
							UserId: testUserId,
						},
					},
				},
				MetaData: strconv.Itoa(i),
			})
		} else {
			batch = append(batch, Message{
				Data: &types.ListUnSubscribeRequest{
					ListId: int64(listId),
					Subscribers: []types.ListUnSubscriber{
						{
							Email:  testEmail,
							UserId: testUserId,
						},
					},
				},
				MetaData: strconv.Itoa(i),
			})
		}
	}
	return batch
}

func testListUnSubscribeHandler(failCnt int, rateLimitCnt int) (*listUnSubscribeHandler, *fakeTransport) {
	transport := NewFakeTransport(failCnt, rateLimitCnt)
	httpClient := http.Client{}
	httpClient.Transport = transport
	lists := api.NewListsApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewListUnSubscribeBatchHandler(lists, &logger.Noop{})
	return handler.(*listUnSubscribeHandler), transport
}
