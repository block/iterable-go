package batch

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/block/iterable-go/api"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptionUpdateBatchHandler_ProcessBatch(t *testing.T) {
	tests := []struct {
		name           string
		failCnt        int
		rateLimitCnt   int
		expectedErr    bool
		expectedRetry  bool
		expectedResLen int
	}{
		{
			name:           "Success",
			failCnt:        0,
			rateLimitCnt:   0,
			expectedErr:    false,
			expectedRetry:  false,
			expectedResLen: 10,
		},
		{
			name:           "Fail",
			failCnt:        1,
			rateLimitCnt:   0,
			expectedErr:    true,
			expectedRetry:  true,
			expectedResLen: 0,
		},
		{
			name:           "RateLimit",
			failCnt:        0,
			rateLimitCnt:   1,
			expectedErr:    true,
			expectedRetry:  true,
			expectedResLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := testSubscriptionUpdateHandler(tt.failCnt, tt.rateLimitCnt)
			batch := generateUserSubUpdateTestBatchMessages(10)
			res, err, retry := handler.ProcessBatch(batch)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResLen, len(res))
			assert.Equal(t, tt.expectedRetry, retry)
		})
	}
}

func TestSubscriptionUpdateBatchHandler_ProcessBatch_DuplicateEmail(t *testing.T) {
	handler := testSubscriptionUpdateHandler(0, 0)
	batch := []Message{
		{
			Data: &types.UserUpdateSubscriptionsRequest{
				Email: testEmail,
			},
		},
		{
			Data: &types.UserUpdateSubscriptionsRequest{
				Email: testEmail,
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
}

func TestSubscriptionUpdateBatchHandler_ProcessBatch_DuplicateId(t *testing.T) {
	handler := testSubscriptionUpdateHandler(0, 0)
	batch := []Message{
		{
			Data: &types.UserUpdateSubscriptionsRequest{
				UserId: testUserId,
			},
		},
		{
			Data: &types.UserUpdateSubscriptionsRequest{
				UserId: testUserId,
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
}

func TestSubscriptionUpdateBatchHandler_ProcessBatch_InvalidData(t *testing.T) {
	handler := testSubscriptionUpdateHandler(0, 0)
	batch := []Message{
		{
			Data: "invalid",
		},
	}
	res, err, retry := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, false, retry)

	assert.Error(t, res[0].Error)
}

func TestSubscriptionUpdateBatchHandler_ProcessOne(t *testing.T) {
	handler := testSubscriptionUpdateHandler(0, 0)
	batch := generateUserSubUpdateTestBatchMessages(10)

	res := make([]Response, 0, len(batch))
	for _, req := range batch {
		res = append(res, handler.ProcessOne(req))
	}

	assert.Equal(t, 10, len(res))
}

func TestSubscriptionUpdateBatchHandler_ProcessOne_InvalidData(t *testing.T) {
	handler := testSubscriptionUpdateHandler(0, 0)
	req := Message{
		Data: "invalid",
	}
	res := handler.ProcessOne(req)

	assert.Error(t, res.Error)
}

func generateUserSubUpdateTestBatchMessages(cnt int) []Message {
	var batch []Message
	for i := range cnt {
		batch = append(batch, Message{
			Data: &types.UserUpdateSubscriptionsRequest{
				Email:                    "test" + strconv.Itoa(i) + "@test.com",
				SubscribedMessageTypeIds: []int64{1, 2, 3},
			},
			MetaData: strconv.Itoa(i),
		})
	}
	return batch
}

func testSubscriptionUpdateHandler(failCnt int, rateLimitCnt int) *subscriptionUpdateHandler {
	transport := NewFakeTransport(failCnt, rateLimitCnt)
	httpClient := http.Client{}
	httpClient.Transport = transport
	lists := api.NewUsersApi("test", &httpClient, &logger.Noop{})
	handler := NewSubscriptionUpdateHandler(lists, &logger.Noop{})
	return handler.(*subscriptionUpdateHandler)
}
