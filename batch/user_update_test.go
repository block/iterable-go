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

func TestUserUpdateBatchHandler_ProcessBatch(t *testing.T) {
	tests := []struct {
		name               string
		failCnt            int
		rateLimitCnt       int
		contentTooLargeCnt int
		expectedErr        bool
		expectedRetry      bool
		expectedResLen     int
	}{
		{
			name:               "Success",
			failCnt:            0,
			rateLimitCnt:       0,
			contentTooLargeCnt: 0,
			expectedErr:        false,
			expectedRetry:      false,
			expectedResLen:     10,
		},
		{
			name:               "Fail",
			failCnt:            1,
			rateLimitCnt:       0,
			contentTooLargeCnt: 0,
			expectedErr:        true,
			expectedRetry:      true,
			expectedResLen:     0,
		},
		{
			name:               "RateLimit",
			failCnt:            0,
			rateLimitCnt:       1,
			contentTooLargeCnt: 0,
			expectedErr:        true,
			expectedRetry:      true,
			expectedResLen:     0,
		},
		{
			name:               "ContentTooLarge",
			failCnt:            0,
			rateLimitCnt:       0,
			contentTooLargeCnt: 1,
			expectedErr:        false,
			expectedRetry:      true,
			expectedResLen:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := testUserUpdateHandler(tt.failCnt, tt.rateLimitCnt, tt.contentTooLargeCnt)
			batch := generateUserUpdateTestBatchMessages(10)
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

func generateUserUpdateTestBatchMessages(cnt int) []Message {
	var batch []Message
	for i := range cnt {
		batch = append(batch, Message{
			Data: &types.BulkUpdateUser{
				Email:      "test" + strconv.Itoa(i) + "@test.com",
				UserId:     "id-" + strconv.Itoa(i),
				DataFields: map[string]any{},
			},
			MetaData: strconv.Itoa(i),
		})
	}
	return batch
}

func testUserUpdateHandler(failCnt int, rateLimitCnt int, contentTooLargeCnt int) *userUpdateHandler {
	transport := NewFakeTransport(failCnt, rateLimitCnt)
	transport.SetContentTooLargeCount(contentTooLargeCnt)
	httpClient := http.Client{}
	httpClient.Transport = transport
	lists := api.NewUsersApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewUserUpdateHandler(lists, &logger.Noop{})
	return handler.(*userUpdateHandler)
}
