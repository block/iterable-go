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

func TestListSubscribeBatchHandler_ProcessBatch(t *testing.T) {
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
			name:             "Batch Request: 500 no body",
			batchSize:        10,
			resStatusCode:    500,
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: 500 with body",
			batchSize:        10,
			resStatusCode:    500,
			resBody:          []byte(`{}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: 500 + ITERABLE_InvalidList",
			batchSize:        10,
			resStatusCode:    500,
			resBody:          []byte(`{"msg":"123 error.lists.invalidListId abc"}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 10,
			expectMsgRetry:   0,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: RateLimit",
			batchSize:        10,
			resStatusCode:    429,
			resBody:          []byte(`{}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: ContentTooLarge",
			batchSize:        10,
			resStatusCode:    413,
			resBody:          []byte(`{}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: Conflict",
			batchSize:        10,
			resStatusCode:    409,
			resBody:          []byte(`{}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: Request Timeout",
			batchSize:        10,
			resStatusCode:    408,
			resBody:          []byte(`{}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: Forbidden",
			batchSize:        10,
			resStatusCode:    403,
			resBody:          []byte(`{}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 10,
			expectMsgRetry:   0,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: 400",
			batchSize:        10,
			resStatusCode:    400,
			resBody:          []byte(`{}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 10,
			expectMsgRetry:   0,
			expectMsgError:   10,
			expectMsgNoError: 0,
		},
		{
			name:             "Batch Request: 400 + ITERABLE_InvalidList",
			batchSize:        10,
			resStatusCode:    400,
			resBody:          []byte(`{"msg":"123 error.lists.invalidListId abc"}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusPartialSuccess{},
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
			resBody:          []byte(`{}`),
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
			handler := testListSubscribeHandler(transport)
			batch := generateListSubTestBatch(tt.batchSize, false, 0)
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

func TestListSubscribeBatchHandler_ProcessBatch_DuplicateEmail(t *testing.T) {
	batch := []Message{
		{
			Data: &types.ListSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListSubscriber{
					{Email: testEmail},
				},
			},
		},
		{
			Data: &types.ListSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListSubscriber{
					{Email: testEmail},
				},
			},
		},
	}
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testListSubscribeHandler(transport)

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

func TestListSubscribeBatchHandler_ProcessBatch_DuplicateId(t *testing.T) {
	batch := []Message{
		{
			Data: &types.ListSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListSubscriber{
					{
						UserId: testUserId,
					},
				},
			},
		},
		{
			Data: &types.ListSubscribeRequest{
				ListId: testListId,
				Subscribers: []types.ListSubscriber{
					{
						UserId: testUserId,
					},
				},
			},
		},
	}
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testListSubscribeHandler(transport)

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

func TestListSubscribeBatchHandler_ProcessBatch_PartialSuccess(t *testing.T) {
	list1Res := types.ListSubscribeResponse{
		FailedUpdates: types.FailedUpdates{
			ConflictEmails:  []string{"email1@example.com"},
			ConflictUserIds: []string{"email1"},
		},
	}
	list2Res := types.ListSubscribeResponse{
		FailedUpdates: types.FailedUpdates{
			ForgottenEmails:    []string{"email2@example.com"},
			ForgottenUserIds:   []string{"email2"},
			InvalidDataEmails:  []string{"email3@example.com"},
			InvalidDataUserIds: []string{"email3"},
			InvalidEmails:      []string{"email4@example.com"},
			InvalidUserIds:     []string{"email4"},
		},
	}
	list3Res := types.ListSubscribeResponse{
		FailedUpdates: types.FailedUpdates{
			NotFoundEmails:  []string{"email5@example.com"},
			NotFoundUserIds: []string{"email5"},
		},
	}
	list4Res := types.ListSubscribeResponse{}
	req := []Message{
		{Data: "invalid"},
		{Data: &types.ListSubscribeRequest{
			ListId:      1,
			Subscribers: []types.ListSubscriber{{Email: "email1@example.com"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      1,
			Subscribers: []types.ListSubscriber{{UserId: "email1"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      2,
			Subscribers: []types.ListSubscriber{{Email: "email2@example.com"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      2,
			Subscribers: []types.ListSubscriber{{UserId: "email2"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      2,
			Subscribers: []types.ListSubscriber{{Email: "email3@example.com"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      2,
			Subscribers: []types.ListSubscriber{{UserId: "email3"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      2,
			Subscribers: []types.ListSubscriber{{Email: "email4@example.com"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      2,
			Subscribers: []types.ListSubscriber{{UserId: "email4"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      3,
			Subscribers: []types.ListSubscriber{{Email: "email5@example.com"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      3,
			Subscribers: []types.ListSubscriber{{UserId: "email5"}},
		}},

		// Successful
		{Data: &types.ListSubscribeRequest{
			ListId:      3,
			Subscribers: []types.ListSubscriber{{Email: "email6@example.com"}},
		}},
		{Data: &types.ListSubscribeRequest{
			ListId:      4,
			Subscribers: []types.ListSubscriber{{UserId: "email6"}},
		}},
	}

	transport := NewFakeTransport(0, 0)

	// expect 4 calls, because there are 4 distinct list ids
	data1, _ := json.Marshal(list1Res)
	data2, _ := json.Marshal(list2Res)
	data3, _ := json.Marshal(list3Res)
	data4, _ := json.Marshal(list4Res)
	transport.AddResponseQueue(200, data1)
	transport.AddResponseQueue(200, data2)
	transport.AddResponseQueue(200, data3)
	transport.AddResponseQueue(200, data4)

	handler := testListSubscribeHandler(transport)
	res, err := handler.ProcessBatch(req)

	assert.Equal(t, 4, transport.reqCnt)
	assert.NoError(t, err)
	assert.IsType(t, StatusPartialSuccess{}, res)
	assert.Equal(t, len(req), len(res.response()))

	msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res.response())
	assert.Equal(t, 0, msgRetry)
	assert.Equal(t, 11, msgNoRetry)
	assert.Equal(t, 11, msgErr)
	assert.Equal(t, 2, msgNoErr)
}

func TestListSubscribeBatchHandler_ProcessOne(t *testing.T) {
	tests := []struct {
		name           string
		resStatusCode  int
		resBody        []byte
		expectApiCalls int
		expectErr      bool
		expectRetry    bool
	}{
		{
			name:           "Success",
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
			handler := testListSubscribeHandler(transport)

			req := generateListSubTestBatch(1, false, 0)[0]
			res := handler.ProcessOne(req)

			assert.ErrorIs(t, res.Error, ErrProcessOneNotAllowed)
			assert.ErrorIs(t, res.Error, ErrClientMustRetryBatchApiErr)
			var apiErr *iterable_errors.ApiError
			ok := errors.As(res.Error, &apiErr)
			require.True(t, ok)
			assert.Equal(t, 500, apiErr.HttpStatusCode)
			assert.True(t, res.Retry)
			assert.Equal(t, 0, transport.reqCnt)
		})
	}
}

func generateListSubTestBatch(cnt int, uniqueIds bool, noEmailCnt int) []Message {
	var batch []Message
	listId := testListId
	for i := range cnt {
		if uniqueIds {
			listId = i
		}
		if i < noEmailCnt {
			batch = append(batch, Message{
				Data: &types.ListSubscribeRequest{
					ListId: int64(listId),
					Subscribers: []types.ListSubscriber{
						{
							UserId: testUserId,
						},
					},
				},
				MetaData: strconv.Itoa(i),
			})
		} else {
			batch = append(batch, Message{
				Data: &types.ListSubscribeRequest{
					ListId: int64(listId),
					Subscribers: []types.ListSubscriber{
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

func testListSubscribeHandler(transport http.RoundTripper) *listSubscribeHandler {
	//transport := NewFakeTransport(failCnt, rateLimitCnt)
	httpClient := http.Client{}
	httpClient.Transport = transport
	lists := api.NewListsApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewListSubscribeHandler(lists, &logger.Noop{})
	return handler.(*listSubscribeHandler)
}
