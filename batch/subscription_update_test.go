package batch

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/rate"
	"github.com/block/iterable-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionUpdateBatchHandler_ProcessBatch(t *testing.T) {
	tests := []struct {
		name             string
		batchSize        int
		invalidSize      int
		resStatusCode    int
		resBody          []byte
		expectApiCalls   int
		expectErr        bool
		expectStatus     ProcessBatchResponse
		expectMsgNoRetry int
		expectMsgRetry   int
		expectMsgError   int
		expectMsgNoError int
	}{
		{
			name:             "Empty",
			batchSize:        0,
			resStatusCode:    200,
			expectApiCalls:   0,
			expectErr:        false,
			expectStatus:     StatusSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   0,
			expectMsgError:   0,
			expectMsgNoError: 0,
		},
		{
			name:             "Success",
			batchSize:        10,
			resStatusCode:    200,
			resBody:          []byte(`{"successCount":10}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   0,
			expectMsgError:   0,
			expectMsgNoError: 10,
		},
		{
			name:             "Batch Request: 500",
			batchSize:        10,
			resStatusCode:    500,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusRetryBatch{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: RateLimit",
			batchSize:        10,
			resStatusCode:    429,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusRetryBatch{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: ContentTooLarge",
			batchSize:        10,
			resStatusCode:    413,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusRetryIndividual{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: Conflict",
			batchSize:        10,
			resStatusCode:    409,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusRetryIndividual{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: Request Timeout",
			batchSize:        10,
			resStatusCode:    408,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusRetryIndividual{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: Forbidden",
			batchSize:        10,
			resStatusCode:    403,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusCannotRetry{},
			expectMsgNoRetry: 10,
			expectMsgRetry:   0,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: 400",
			batchSize:        10,
			resStatusCode:    400,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusCannotRetry{},
			expectMsgNoRetry: 10,
			expectMsgRetry:   0,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: 200 + invalid messages",
			batchSize:        10,
			invalidSize:      5,
			resStatusCode:    200,
			resBody:          []byte(`{"successCount":10}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 5,
			expectMsgRetry:   0,
			expectMsgError:   5,
			expectMsgNoError: 10,
		},
		{
			name:             "all messages are invalid",
			invalidSize:      5,
			resStatusCode:    200,
			resBody:          []byte(`{"successCount":5}`), // never called
			expectApiCalls:   0,
			expectErr:        false,
			expectStatus:     StatusCannotRetry{},
			expectMsgNoRetry: 5,
			expectMsgRetry:   0,
			expectMsgError:   5,
			expectMsgNoError: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewFakeTransport(0, 0)
			transport.AddResponseQueue(tt.resStatusCode, tt.resBody)
			handler := testSubscriptionUpdateHandler(transport)
			batch := generateUserSubUpdateTestBatch(tt.batchSize)
			for range tt.invalidSize {
				batch = append(batch, Message{
					Data: "invalid",
				})
			}

			res, err := handler.ProcessBatch(batch)

			assert.Equal(t, tt.expectApiCalls, transport.reqCnt)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectStatus != nil {
				assert.IsType(t, tt.expectStatus, res)
			}

			msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res.response())

			assert.Equal(t, tt.expectMsgRetry, msgRetry)
			assert.Equal(t, tt.expectMsgNoRetry, msgNoRetry)
			assert.Equal(t, tt.expectMsgError, msgErr)
			assert.Equal(t, tt.expectMsgNoError, msgNoErr)
		})
	}
}

func TestSubscriptionUpdateBatchHandler_ProcessBatch_DuplicateEmail(t *testing.T) {
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testSubscriptionUpdateHandler(transport)

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
	res, err := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.IsType(t, StatusSuccess{}, res)
	assert.Equal(t, 2, len(res.response()))

	msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res.response())
	assert.Equal(t, 0, msgNoRetry)
	assert.Equal(t, 0, msgRetry)
	assert.Equal(t, 2, msgNoErr)
	assert.Equal(t, 0, msgErr)
}

func TestSubscriptionUpdateBatchHandler_ProcessBatch_DuplicateId(t *testing.T) {
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testSubscriptionUpdateHandler(transport)

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
	res, err := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.IsType(t, StatusSuccess{}, res)
	assert.Equal(t, 2, len(res.response()))

	msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res.response())
	assert.Equal(t, 0, msgNoRetry)
	assert.Equal(t, 0, msgRetry)
	assert.Equal(t, 2, msgNoErr)
	assert.Equal(t, 0, msgErr)
}

func TestSubscriptionUpdateBatchHandle_ProcessBatch_PartialSuccess(t *testing.T) {
	res := types.BulkUserUpdateSubscriptionsResponse{
		SuccessCount:        0,
		FailCount:           4,
		InvalidEmails:       []string{"email1@example.com"},
		InvalidUserIds:      []string{"email1"},
		ValidEmailFailures:  []string{"email2@example.com"},
		ValidUserIdFailures: []string{"email2"},
	}
	data, _ := json.Marshal(res)
	req := []Message{
		{Data: "invalid"},
		{Data: &types.UserUpdateSubscriptionsRequest{
			Email: "email1@example.com",
		}},
		{Data: &types.UserUpdateSubscriptionsRequest{
			UserId: "email1",
		}},
		{Data: &types.UserUpdateSubscriptionsRequest{
			Email: "email2@example.com",
		}},
		{Data: &types.UserUpdateSubscriptionsRequest{
			UserId: "email2",
		}},

		// Successful
		{Data: &types.UserUpdateSubscriptionsRequest{Email: "email6@example.com"}},
		{Data: &types.UserUpdateSubscriptionsRequest{UserId: "email6"}},
	}

	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, data)
	handler := testSubscriptionUpdateHandler(transport)
	res2, err := handler.ProcessBatch(req)

	assert.Equal(t, 1, transport.reqCnt)
	assert.NoError(t, err)
	assert.IsType(t, StatusPartialSuccess{}, res2)
	assert.Equal(t, len(req), len(res2.response()))

	msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res2.response())

	assert.Equal(t, 0, msgRetry)
	assert.Equal(t, 5, msgNoRetry)
	assert.Equal(t, 5, msgErr)
	assert.Equal(t, 2, msgNoErr)
}

func TestSubscriptionUpdateBatchHandler_ProcessOne(t *testing.T) {
	tests := []struct {
		name           string
		resStatusCode  int
		resBody        []byte
		expectApiCalls int
		expectErr      bool
		expectRetry    bool
	}{
		{
			name:           "status:200",
			resStatusCode:  200,
			resBody:        []byte(`{"code":"200"}`),
			expectApiCalls: 1,
		},
		{
			name:           "status:0",
			resStatusCode:  0,
			resBody:        []byte(`{}`),
			expectApiCalls: 1,
			expectErr:      true,
			expectRetry:    true,
		},
		{
			name:           "status:500",
			resStatusCode:  500,
			resBody:        []byte(`{}`),
			expectApiCalls: 1,
			expectErr:      true,
			expectRetry:    true,
		},
		{
			name:           "status:408",
			resStatusCode:  408,
			resBody:        []byte(`{}`),
			expectApiCalls: 1,
			expectErr:      true,
			expectRetry:    true,
		},
		{
			name:           "status:429",
			resStatusCode:  408,
			resBody:        []byte(`{}`),
			expectApiCalls: 1,
			expectErr:      true,
			expectRetry:    true,
		},
		{
			name:           "status:409",
			resStatusCode:  409,
			resBody:        []byte(`{}`),
			expectApiCalls: 1,
			expectErr:      true,
			expectRetry:    false,
		},
		{
			name:           "status:400",
			resStatusCode:  400,
			resBody:        []byte(`{}`),
			expectApiCalls: 1,
			expectErr:      true,
			expectRetry:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewFakeTransport(0, 0)
			transport.AddResponseQueue(tt.resStatusCode, tt.resBody)
			handler := testSubscriptionUpdateHandler(transport)

			req := generateUserSubUpdateTestOne()
			res := handler.ProcessOne(req)

			assert.Equal(t, tt.expectApiCalls, transport.reqCnt)
			if tt.expectErr {
				assert.Error(t, res.Error)
				assert.Equal(t, tt.expectRetry, res.Retry)
			} else {
				assert.NoError(t, res.Error)
				assert.False(t, res.Retry)
			}
		})
	}
}

func TestSubscriptionUpdateBatchHandler_ProcessOne_InvalidData(t *testing.T) {
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testSubscriptionUpdateHandler(transport)
	req := Message{
		Data: "invalid",
	}
	res := handler.ProcessOne(req)

	assert.Equal(t, 0, transport.reqCnt)
	assert.Error(t, res.Error)
	assert.ErrorIs(t, res.Error, ErrInvalidDataType)
	assert.ErrorIs(t, res.Error, ErrClientValidationApiErr)
	var apiErr *iterable_errors.ApiError
	ok := errors.As(res.Error, &apiErr)
	require.True(t, ok)
	assert.Equal(t, 400, apiErr.HttpStatusCode)
	assert.False(t, res.Retry)
}

func generateUserSubUpdateTestBatch(cnt int) []Message {
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

func generateUserSubUpdateTestOne() Message {
	return generateUserSubUpdateTestBatch(1)[0]
}

func testSubscriptionUpdateHandler(transport http.RoundTripper) *subscriptionUpdateHandler {
	httpClient := http.Client{}
	httpClient.Transport = transport
	lists := api.NewUsersApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewSubscriptionUpdateHandler(lists, &logger.Noop{})
	return handler.(*subscriptionUpdateHandler)
}
