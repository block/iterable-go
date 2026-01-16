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

func TestEventTrackBatchHandler_ProcessBatch(t *testing.T) {
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
			name:             "Batch Request: 400 + ITERABLE_FieldTypeMismatchErrStr",
			batchSize:        10,
			resStatusCode:    400,
			resBody:          []byte(`{"code":"RequestFieldsTypesMismatched"}`),
			expectApiCalls:   1,
			expectErr:        false,
			expectStatus:     StatusRetryIndividual{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   10,
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
			handler := testEventTrackHandler(transport)
			batch := generateEventTrackTestBatch(tt.batchSize)
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

func TestEventTrackBatchHandler_ProcessBatch_DisallowedEventNames(t *testing.T) {
	tests := []struct {
		name             string
		batch            []Message
		disallowedEvents []string
		expectApiCalls   int
		expectStatus     ProcessBatchResponse
		expectMsgNoRetry int
		expectMsgRetry   int
		expectMsgError   int
		expectMsgNoError int
	}{
		{
			name: "Single disallowed event with email",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						Email:     "test1@test.com",
						EventName: "disallowedEvent",
					},
				},
				{
					Data: &types.EventTrackRequest{
						Email:     "test2@test.com",
						EventName: "allowedEvent",
					},
				},
			},
			disallowedEvents: []string{"disallowedEvent"},
			expectApiCalls:   1,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 1,
			expectMsgRetry:   0,
			expectMsgError:   1,
			expectMsgNoError: 1,
		},
		{
			name: "Single disallowed event with userId",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						UserId:    "user1",
						EventName: "disallowedEvent",
					},
				},
				{
					Data: &types.EventTrackRequest{
						UserId:    "user2",
						EventName: "allowedEvent",
					},
				},
			},
			disallowedEvents: []string{"disallowedEvent"},
			expectApiCalls:   1,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 1,
			expectMsgRetry:   0,
			expectMsgError:   1,
			expectMsgNoError: 1,
		},
		{
			name: "Multiple disallowed events mixed identifiers",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						Email:     "test1@test.com",
						EventName: "disallowedEvent1",
					},
				},
				{
					Data: &types.EventTrackRequest{
						UserId:    "user1",
						EventName: "disallowedEvent2",
					},
				},
				{
					Data: &types.EventTrackRequest{
						Email:     "test2@test.com",
						EventName: "allowedEvent",
					},
				},
			},
			disallowedEvents: []string{"disallowedEvent1", "disallowedEvent2"},
			expectApiCalls:   1,
			expectStatus:     StatusPartialSuccess{},
			expectMsgNoRetry: 2,
			expectMsgRetry:   0,
			expectMsgError:   2,
			expectMsgNoError: 1,
		},
		{
			name: "No disallowed events",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						Email:     "test1@test.com",
						EventName: "allowedEvent1",
					},
				},
				{
					Data: &types.EventTrackRequest{
						UserId:    "user1",
						EventName: "allowedEvent2",
					},
				},
			},
			disallowedEvents: []string{},
			expectApiCalls:   1,
			expectStatus:     StatusSuccess{},
			expectMsgNoRetry: 0,
			expectMsgRetry:   0,
			expectMsgError:   0,
			expectMsgNoError: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewFakeTransport(0, 0)
			transport.SetDisallowedEventNames(tt.disallowedEvents)
			handler := testEventTrackHandler(transport)

			res, err := handler.ProcessBatch(tt.batch)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectApiCalls, transport.reqCnt)

			if tt.expectStatus != nil {
				assert.IsType(t, tt.expectStatus, res)
			}

			msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res.response())

			assert.Equal(t, tt.expectMsgRetry, msgRetry)
			assert.Equal(t, tt.expectMsgNoRetry, msgNoRetry)
			assert.Equal(t, tt.expectMsgError, msgErr)
			assert.Equal(t, tt.expectMsgNoError, msgNoErr)

			for _, r := range res.response() {
				if r.Error != nil {
					assert.ErrorIs(t, r.Error, ErrDisallowedEventName)
					assert.ErrorIs(t, r.Error, ErrServerValidationApiErr)

					var apiErr *iterable_errors.ApiError
					ok := errors.As(r.Error, &apiErr)
					assert.True(t, ok)
					assert.Equal(t, 400, apiErr.HttpStatusCode)
				}
			}
		})
	}
}

func TestEventTrackBatchHandler_ProcessBatch_FieldTypeMismatch(t *testing.T) {
	tests := []struct {
		name           string
		batchSize      int
		resStatusCode  int
		resBody        []byte
		expectApiCalls int
		expectStatus   ProcessBatchResponse
		expectFields   types.MismatchedFieldsParams
	}{
		{
			name:           "400 with empty validationErrors",
			batchSize:      1,
			resStatusCode:  400,
			resBody:        []byte(`{"code":"RequestFieldsTypesMismatched"}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields:   types.MismatchedFieldsParams{},
		},
		{
			name:           "500 with empty validationErrors",
			batchSize:      1,
			resStatusCode:  500,
			resBody:        []byte(`{"code":"RequestFieldsTypesMismatched"}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields:   types.MismatchedFieldsParams{},
		},
		{
			name:          "400 with non-empty validationErrors",
			batchSize:     1,
			resStatusCode: 400,
			resBody: []byte(`{
				"code":"RequestFieldsTypesMismatched",
				"msg": "boo",
				"params": {
					"validationErrors": {
						"my_project_etl:::soc_signup_at": {
							"incomingTypes": ["string", "keyword"],
							"expectedType": "date",
							"category": "user",
							"offendingValue": "2020-04-13 14:04:09.000",
							"_type": "UnexpectedType"
						}
					}
				}
			}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields: types.MismatchedFieldsParams{
				ValidationErrors: types.MismatchedFieldsErrors{
					"my_project_etl:::soc_signup_at": types.MismatchedFieldError{
						IncomingTypes:  []string{"string", "keyword"},
						ExpectedType:   "date",
						Category:       "user",
						OffendingValue: "2020-04-13 14:04:09.000",
						Type:           "UnexpectedType",
					},
				},
			},
		},
		{
			name:          "400 with non-empty validationErrors with extra fields",
			batchSize:     1,
			resStatusCode: 400,
			resBody: []byte(`{
				"code":"RequestFieldsTypesMismatched",
				"msg": "boo",
				"params": {
					"unknown": 1,
					"validationErrors": {
						"my_project_etl:::soc_signup_at": {
							"incomingTypes": ["string", "keyword"],
							"expectedType": "date",
							"unknown": 1,
							"invalid": "a"
						}
					}
				}
			}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields: types.MismatchedFieldsParams{
				ValidationErrors: types.MismatchedFieldsErrors{
					"my_project_etl:::soc_signup_at": types.MismatchedFieldError{
						IncomingTypes: []string{"string", "keyword"},
						ExpectedType:  "date",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewFakeTransport(0, 0)
			transport.AddResponseQueue(tt.resStatusCode, tt.resBody)
			handler := testEventTrackHandler(transport)
			batch := generateEventTrackTestBatch(tt.batchSize)

			res, err := handler.ProcessBatch(batch)
			assert.Equal(t, tt.expectApiCalls, transport.reqCnt)
			assert.NoError(t, err)

			assert.IsType(t, tt.expectStatus, StatusRetryIndividual{})

			for _, r := range res.response() {
				assert.True(t, r.Retry)
				assert.Error(t, r.Error)
				assert.NotNil(t, r.OriginalReq)

				allErr := Unwrap(r.Error)
				assert.Equal(t, 2, len(allErr))

				var err1 *ErrFieldTypeMismatchType
				ok1 := errors.Is(r.Error, ErrFieldTypeMismatch)
				ok2 := errors.As(r.Error, &err1)
				assert.True(t, ok1)
				assert.True(t, ok2)
				assert.NotNil(t, err1)
				ok := assert.ObjectsAreEqualValues(tt.expectFields, err1.MismatchedFields())
				assert.True(t, ok)

				var err2 *iterable_errors.ApiError
				ok1 = errors.Is(r.Error, ErrApiError)
				ok2 = errors.As(r.Error, &err2)
				assert.True(t, ok1)
				assert.True(t, ok2)
				assert.NotNil(t, err2)
				assert.Equal(t, tt.resStatusCode, err2.HttpStatusCode)

			}
		})
	}
}

func TestEventTrackBatchHandler_ProcessBatch_DuplicateEvent(t *testing.T) {
	batch := []Message{
		{
			Data: &types.EventTrackRequest{
				EventName: "testEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				EventName: "testEvent",
			},
		},
	}
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testEventTrackHandler(transport)

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

func TestEventTrackBatchHandler_ProcessBatch_PartialSuccess(t *testing.T) {
	res := types.BulkEventsResponse{
		DisallowedEventNames: []string{"disallowedEvent1", "disallowedEvent2"},
		FilteredOutFields:    []string{},
		CreatedFields:        []string{},
		FailedUpdates: types.FailedEventUpdates{
			InvalidEmails:    []string{"email1@example.com"},
			InvalidUserIds:   []string{"email1"},
			NotFoundEmails:   []string{"email2@example.com"},
			NotFoundUserIds:  []string{"email2"},
			ForgottenEmails:  []string{"email3@example.com"},
			ForgottenUserIds: []string{"email3"},
		},
	}
	req := []Message{
		{Data: "invalid"},
		{
			Data: &types.EventTrackRequest{
				Email:     "email0@example.com",
				EventName: "disallowedEvent1",
			},
		},
		{
			Data: &types.EventTrackRequest{
				UserId:    "email0",
				EventName: "disallowedEvent2",
			},
		},
		{
			Data: &types.EventTrackRequest{
				Email:     "email1@example.com",
				EventName: "allowedEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				UserId:    "email1",
				EventName: "allowedEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				Email:     "email2@example.com",
				EventName: "allowedEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				UserId:    "email2",
				EventName: "allowedEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				Email:     "email3@example.com",
				EventName: "allowedEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				UserId:    "email3",
				EventName: "allowedEvent",
			},
		},

		// Successful
		{
			Data: &types.EventTrackRequest{
				Email:     "email4@example.com",
				EventName: "allowedEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				UserId:    "email4",
				EventName: "allowedEvent",
			},
		},
	}

	transport := NewFakeTransport(0, 0)

	data, _ := json.Marshal(res)
	transport.AddResponseQueue(200, data)
	transport.SetDisallowedEventNames([]string{"disallowedEvent1", "disallowedEvent2"})

	handler := testEventTrackHandler(transport)
	res2, err := handler.ProcessBatch(req)

	assert.Equal(t, 1, transport.reqCnt)
	assert.NoError(t, err)
	assert.IsType(t, StatusPartialSuccess{}, res2)
	assert.Equal(t, len(req), len(res2.response()))

	msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res2.response())
	assert.Equal(t, 0, msgRetry)
	assert.Equal(t, 9, msgNoRetry)
	assert.Equal(t, 9, msgErr)
	assert.Equal(t, 2, msgNoErr)
}

func TestEventTrackBatchHandler_ProcessOne(t *testing.T) {
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
			handler := testEventTrackHandler(transport)

			req := generateEventTrackTestBatch(1)[0]
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

func TestEventTrackBatchHandler_ProcessOne_InvalidData(t *testing.T) {
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testEventTrackHandler(transport)
	req := Message{
		Data: "invalid",
	}
	res := handler.ProcessOne(req)

	assert.Equal(t, 0, transport.reqCnt)
	assert.Error(t, res.Error)
	assert.ErrorIs(t, res.Error, ErrInvalidDataType)
	assert.False(t, res.Retry)
}

func TestEventTrackBatchHandler_ProcessOne_DisallowedEvent(t *testing.T) {
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{"msg":"event x is disallowed from tracking."}`))

	handler := testEventTrackHandler(transport)
	req := Message{
		Data: &types.EventTrackRequest{
			Email:     "test1@test.com",
			EventName: "disallowedEvent",
		}}
	res := handler.ProcessOne(req)

	assert.Equal(t, 1, transport.reqCnt)
	assert.Error(t, res.Error)
	assert.ErrorIs(t, res.Error, ErrDisallowedEventName)
	assert.ErrorIs(t, res.Error, ErrServerValidationApiErr)
	var apiErr *iterable_errors.ApiError
	ok := errors.As(res.Error, &apiErr)
	require.True(t, ok)
	assert.Equal(t, 400, apiErr.HttpStatusCode)
	assert.False(t, res.Retry)
}

func TestEventTrackBatchHandler_ProcessOne_FieldTypeMismatch(t *testing.T) {
	tests := []struct {
		name           string
		batchSize      int
		resStatusCode  int
		resBody        []byte
		expectApiCalls int
		expectStatus   ProcessBatchResponse
		expectFields   types.MismatchedFieldsParams
	}{
		{
			name:           "400 with empty validationErrors",
			batchSize:      1,
			resStatusCode:  400,
			resBody:        []byte(`{"code":"RequestFieldsTypesMismatched"}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields:   types.MismatchedFieldsParams{},
		},
		{
			name:           "500 with empty validationErrors",
			batchSize:      1,
			resStatusCode:  500,
			resBody:        []byte(`{"code":"RequestFieldsTypesMismatched"}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields:   types.MismatchedFieldsParams{},
		},
		{
			name:          "400 with non-empty validationErrors",
			batchSize:     1,
			resStatusCode: 400,
			resBody: []byte(`{
				"code":"RequestFieldsTypesMismatched",
				"msg": "boo",
				"params": {
					"validationErrors": {
						"my_project_etl:::soc_signup_at": {
							"incomingTypes": ["string", "keyword"],
							"expectedType": "date",
							"category": "user",
							"offendingValue": "2020-04-13 14:04:09.000",
							"_type": "UnexpectedType"
						}
					}
				}
			}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields: types.MismatchedFieldsParams{
				ValidationErrors: types.MismatchedFieldsErrors{
					"my_project_etl:::soc_signup_at": types.MismatchedFieldError{
						IncomingTypes:  []string{"string", "keyword"},
						ExpectedType:   "date",
						Category:       "user",
						OffendingValue: "2020-04-13 14:04:09.000",
						Type:           "UnexpectedType",
					},
				},
			},
		},
		{
			name:          "400 with non-empty validationErrors with extra fields",
			batchSize:     1,
			resStatusCode: 400,
			resBody: []byte(`{
				"code":"RequestFieldsTypesMismatched",
				"msg": "boo",
				"params": {
					"unknown": 1,
					"validationErrors": {
						"my_project_etl:::soc_signup_at": {
							"incomingTypes": ["string", "keyword"],
							"expectedType": "date",
							"unknown": 1,
							"invalid": "a"
						}
					}
				}
			}`),
			expectApiCalls: 1,
			expectStatus:   StatusRetryIndividual{},
			expectFields: types.MismatchedFieldsParams{
				ValidationErrors: types.MismatchedFieldsErrors{
					"my_project_etl:::soc_signup_at": types.MismatchedFieldError{
						IncomingTypes: []string{"string", "keyword"},
						ExpectedType:  "date",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewFakeTransport(0, 0)
			transport.AddResponseQueue(tt.resStatusCode, tt.resBody)
			handler := testEventTrackHandler(transport)
			batch := generateEventTrackTestBatch(tt.batchSize)

			res, err := handler.ProcessBatch(batch)
			assert.Equal(t, tt.expectApiCalls, transport.reqCnt)
			assert.NoError(t, err)

			assert.IsType(t, tt.expectStatus, StatusRetryIndividual{})

			for _, r := range res.response() {
				assert.True(t, r.Retry)
				assert.Error(t, r.Error)
				assert.NotNil(t, r.OriginalReq)

				allErr := Unwrap(r.Error)
				assert.Equal(t, 2, len(allErr))

				var err1 *ErrFieldTypeMismatchType
				ok1 := errors.Is(r.Error, ErrFieldTypeMismatch)
				ok2 := errors.As(r.Error, &err1)
				assert.True(t, ok1)
				assert.True(t, ok2)
				assert.NotNil(t, err1)
				ok := assert.ObjectsAreEqualValues(tt.expectFields, err1.MismatchedFields())
				assert.True(t, ok)

				var err2 *iterable_errors.ApiError
				ok1 = errors.Is(r.Error, ErrApiError)
				ok2 = errors.As(r.Error, &err2)
				assert.True(t, ok1)
				assert.True(t, ok2)
				assert.NotNil(t, err2)
				assert.Equal(t, tt.resStatusCode, err2.HttpStatusCode)

				assert.True(t, errors.Is(r.Error, ErrServerValidationApiErr))
			}
		})
	}
}

func generateEventTrackTestBatch(cnt int) []Message {
	var batch []Message
	for i := 0; i < cnt; i++ {
		batch = append(batch, Message{
			Data: &types.EventTrackRequest{
				Email:     "testEmail" + strconv.Itoa(i) + "@test.com",
				EventName: "testEvent" + strconv.Itoa(i),
			},
			MetaData: strconv.Itoa(i),
		})
	}
	return batch
}

func testEventTrackHandler(transport http.RoundTripper) *eventTrackHandler {
	httpClient := http.Client{}
	httpClient.Transport = transport
	ev := api.NewEventsApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewEventTrackHandler(ev, &logger.Noop{})
	return handler.(*eventTrackHandler)
}
