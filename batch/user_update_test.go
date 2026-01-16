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

func TestUserUpdateBatchHandler_ProcessBatch(t *testing.T) {
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
			name:             "Batch Request: 400 + RequestFieldsTypesMismatched",
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
			handler := testUserUpdateHandler(transport)
			batch := generateUserUpdateTestBatch(tt.batchSize)
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

func TestUserUpdateBatchHandler_ProcessBatch_PartialSuccess(t *testing.T) {
	res := types.BulkUpdateResponse{
		FailedUpdates: types.FailedUpdates{
			ConflictEmails:     []string{"email1@example.com"},
			ConflictUserIds:    []string{"email1"},
			ForgottenEmails:    []string{"email2@example.com"},
			ForgottenUserIds:   []string{"email2"},
			InvalidDataEmails:  []string{"email3@example.com"},
			InvalidDataUserIds: []string{"email3"},
			InvalidEmails:      []string{"email4@example.com"},
			InvalidUserIds:     []string{"email4"},
			NotFoundEmails:     []string{"email5@example.com"},
			NotFoundUserIds:    []string{"email5"},
		},
	}
	data, _ := json.Marshal(res)
	req := []Message{
		{Data: "invalid"},
		{Data: &types.BulkUpdateUser{Email: "email1@example.com"}},
		{Data: &types.BulkUpdateUser{UserId: "email1"}},
		{Data: &types.BulkUpdateUser{Email: "email2@example.com"}},
		{Data: &types.BulkUpdateUser{UserId: "email2"}},
		{Data: &types.BulkUpdateUser{Email: "email3@example.com"}},
		{Data: &types.BulkUpdateUser{UserId: "email3"}},
		{Data: &types.BulkUpdateUser{Email: "email4@example.com"}},
		{Data: &types.BulkUpdateUser{UserId: "email4"}},
		{Data: &types.BulkUpdateUser{Email: "email5@example.com"}},
		{Data: &types.BulkUpdateUser{UserId: "email5"}},
		// Successful
		{Data: &types.BulkUpdateUser{Email: "email6@example.com"}},
		{Data: &types.BulkUpdateUser{UserId: "email6"}},
	}

	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, data)
	handler := testUserUpdateHandler(transport)
	res2, err := handler.ProcessBatch(req)

	assert.Equal(t, 1, transport.reqCnt)
	assert.NoError(t, err)
	assert.IsType(t, StatusPartialSuccess{}, res2)
	assert.Equal(t, len(req), len(res2.response()))

	msgNoRetry, msgRetry, msgNoErr, msgErr := countResponses(t, res2.response())

	assert.Equal(t, 0, msgRetry)
	assert.Equal(t, 11, msgNoRetry)
	assert.Equal(t, 11, msgErr)
	assert.Equal(t, 2, msgNoErr)
}

func TestUserUpdateBatchHandler_ProcessBatch_FieldTypeMismatch(t *testing.T) {
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
			handler := testUserUpdateHandler(transport)
			batch := generateUserUpdateTestBatch(tt.batchSize)

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

func TestUserUpdateBatchHandler_ProcessOne(t *testing.T) {
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
			handler := testUserUpdateHandler(transport)

			req := generateUserUpdateTestOne()
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

func TestUserUpdateBatchHandler_ProcessOne_InvalidData(t *testing.T) {
	transport := NewFakeTransport(0, 0)
	transport.AddResponseQueue(200, []byte(`{}`))
	handler := testUserUpdateHandler(transport)
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

func TestUserUpdateBatchHandler_ProcessOne_FieldTypeMismatch(t *testing.T) {
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
			handler := testUserUpdateHandler(transport)
			batch := generateUserUpdateTestBatch(tt.batchSize)

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

func generateUserUpdateTestBatch(size int) []Message {
	var batch []Message
	for i := range size {
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

func generateUserUpdateTestOne() Message {
	return generateUserUpdateTestBatch(1)[0]
}

func testUserUpdateHandler(transport http.RoundTripper) *userUpdateHandler {
	httpClient := http.Client{}
	httpClient.Transport = transport
	users := api.NewUsersApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewUserUpdateHandler(users, &logger.Noop{})
	return handler.(*userUpdateHandler)
}

func countResponses(t *testing.T, res []Response) (int, int, int, int) {
	msgNoRetry := 0
	msgRetry := 0
	msgErr := 0
	msgNoErr := 0
	for _, r := range res {
		if r.Error != nil {
			msgErr++
			if r.Retry {
				msgRetry++
			} else {
				msgNoRetry++
			}
		} else {
			msgNoErr++
		}
		assert.NotNil(t, r.OriginalReq)
	}
	return msgNoRetry, msgRetry, msgNoErr, msgErr
}
