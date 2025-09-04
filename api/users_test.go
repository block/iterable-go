package api

import (
	"encoding/json"
	"iterable-go/errors"
	"iterable-go/logger"
	"iterable-go/types"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUsersApi(t *testing.T) {
	client := &http.Client{}
	api := NewUsersApi(testApiKey, client, &logger.Noop{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.api)
	assert.Equal(t, testApiKey, api.api.apiKey)
	assert.Equal(t, client, api.api.httpClient)
}

func TestUsers_UpdateOrCreate(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.UserRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful create",
			request: types.UserRequest{
				Email: "test@example.com",
				DataFields: map[string]interface{}{
					"firstName": "Test",
					"lastName":  "User",
				},
			},
			resBody:   []byte(`{"code": "Success", "msg": "User created"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/update",
			expectRes: &types.PostResponse{Code: "Success", Message: "User created"},
		},
		{
			name: "update existing user",
			request: types.UserRequest{
				Email: "existing@example.com",
				DataFields: map[string]interface{}{
					"updated": true,
				},
			},
			resBody:   []byte(`{"code": "Success", "msg": "User updated"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/update",
			expectRes: &types.PostResponse{Code: "Success", Message: "User updated"},
		},
		{
			name: "malformed",
			request: types.UserRequest{
				Email: "error@example.com",
			},
			resBody:    []byte(`{"code": "Error", "m`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/users/update",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name: "server error",
			request: types.UserRequest{
				Email: "error@example.com",
			},
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/update",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "429",
			request: types.UserRequest{
				Email: "error@example.com",
			},
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/update",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "network error",
			request: types.UserRequest{
				Email: "error@example.com",
			},
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/update",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, &logger.Noop{})

			res, err := api.UpdateOrCreate(tt.request)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, res)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodPost, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())

			if !tt.expectErr {
				var sentRequest types.UserRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestUsers_BulkUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.BulkUpdateRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.BulkUpdateResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful bulk update",
			request: types.BulkUpdateRequest{
				Users: []types.BulkUpdateUser{
					{Email: "user1@example.com"},
					{Email: "user2@example.com"},
				},
			},
			resBody:   []byte(`{"successCount": 2, "failCount": 0}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/bulkUpdate",
			expectRes: &types.BulkUpdateResponse{
				SuccessCount: 2,
				FailCount:    0,
			},
		},
		{
			name: "partial success",
			request: types.BulkUpdateRequest{
				Users: []types.BulkUpdateUser{
					{Email: "valid@example.com"},
					{Email: "invalid@"},
				},
			},
			resBody: []byte(`{
				"successCount": 1,
				"failCount": 1,
				"failedUpdates": {"invalidEmails": ["invalid@"]}
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/bulkUpdate",
			expectRes: &types.BulkUpdateResponse{
				SuccessCount: 1,
				FailCount:    1,
				FailedUpdates: types.FailedUpdates{
					InvalidEmails: []string{"invalid@"},
				},
			},
		},
		{
			name: "server error",
			request: types.BulkUpdateRequest{
				Users: []types.BulkUpdateUser{
					{Email: "valid@example.com"},
					{Email: "invalid@"},
				},
			},
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/bulkUpdate",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "429",
			request: types.BulkUpdateRequest{
				Users: []types.BulkUpdateUser{
					{Email: "valid@example.com"},
					{Email: "invalid@"},
				},
			},
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/bulkUpdate",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "network error",
			request: types.BulkUpdateRequest{
				Users: []types.BulkUpdateUser{
					{Email: "valid@example.com"},
					{Email: "invalid@"},
				},
			},
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/bulkUpdate",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, &logger.Noop{})

			res, err := api.BulkUpdate(tt.request)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, res)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodPost, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())

			if !tt.expectErr {
				var sentRequest types.BulkUpdateRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestUsers_GetByEmail(t *testing.T) {
	testCases := []struct {
		name        string
		email       string
		resBody     []byte
		resCode     int
		resErr      error
		expectUrl   string
		expectRes   *types.User
		expectFound bool
		expectErr   bool
		resErrType  string
	}{
		{
			name:  "user found",
			email: "test@example.com",
			resBody: []byte(`{
				"user": {
					"email": "test@example.com",
					"userId": "123",
					"dataFields": {"firstName": "Test"}
				}
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/getByEmail?email=test%40example.com",
			expectRes: &types.User{
				Email:  "test@example.com",
				UserId: "123",
				DataFields: map[string]interface{}{
					"firstName": "Test",
				},
			},
			expectFound: true,
		},
		{
			name:        "user not found",
			email:       "notfound@example.com",
			resBody:     []byte(`{"user": {}}`),
			resCode:     200,
			expectUrl:   "https://api.iterable.com/api/users/getByEmail?email=notfound%40example.com",
			expectFound: false,
		},
		{
			name:       "server error",
			email:      "error@example.com",
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/getByEmail?email=error%40example.com",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			email:      "error@example.com",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/getByEmail?email=error%40example.com",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			email:      "error@example.com",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/getByEmail?email=error%40example.com",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, &logger.Noop{})

			user, found, err := api.GetByEmail(tt.email)
			if tt.expectErr {
				assert.Error(t, err)
				assert.False(t, found)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectFound, found)
				if tt.expectFound {
					assert.Equal(t, tt.expectRes, user)
				} else {
					assert.Nil(t, user)
				}
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestUsers_GetById(t *testing.T) {
	testCases := []struct {
		name        string
		userId      string
		resBody     []byte
		resCode     int
		resErr      error
		expectUrl   string
		expectRes   *types.User
		expectFound bool
		expectErr   bool
		resErrType  string
	}{
		{
			name:   "user found",
			userId: "123",
			resBody: []byte(`{
				"user": {
					"email": "test@example.com",
					"userId": "123",
					"dataFields": {"firstName": "Test"}
				}
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/byUserId?userId=123",
			expectRes: &types.User{
				Email:  "test@example.com",
				UserId: "123",
				DataFields: map[string]interface{}{
					"firstName": "Test",
				},
			},
			expectFound: true,
		},
		{
			name:        "user not found",
			userId:      "456",
			resBody:     []byte(`{"code": "error.users.noUserWithIdExists", "msg": "User not found"}`),
			resCode:     500,
			expectUrl:   "https://api.iterable.com/api/users/byUserId?userId=456",
			expectErr:   false,
			expectFound: false,
		},
		{
			name:   "user not found 2",
			userId: "456",
			resBody: []byte(`{
				"user": {
					"email": "",
					"userId": ""
				}
			}`),
			resCode:     200,
			expectUrl:   "https://api.iterable.com/api/users/byUserId?userId=456",
			expectErr:   false,
			expectFound: false,
		},
		{
			name:       "server error",
			userId:     "456",
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/byUserId?userId=456",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			userId:     "456",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/byUserId?userId=456",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			userId:     "456",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/byUserId?userId=456",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, &logger.Noop{})

			user, found, err := api.GetById(tt.userId)
			if tt.expectErr {
				assert.Error(t, err)
				assert.False(t, found)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectFound, found)
				if tt.expectFound {
					assert.Equal(t, tt.expectRes, user)
				} else {
					assert.Nil(t, user)
				}
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestUsers_DeleteByEmail(t *testing.T) {
	testCases := []struct {
		name       string
		email      string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name:      "successful delete",
			email:     "delete@example.com",
			resBody:   []byte(`{"code": "Success", "msg": "User deleted"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/delete@example.com",
			expectRes: &types.PostResponse{Code: "Success", Message: "User deleted"},
		},
		{
			name:       "user not found",
			email:      "notfound@example.com",
			resBody:    []byte(`{"code": "NotFound", "msg": "User not found"}`),
			resCode:    404,
			expectUrl:  "https://api.iterable.com/api/users/notfound@example.com",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "server error",
			email:      "error@example.com",
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/error@example.com",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			email:      "error@example.com",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/error@example.com",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			email:      "error@example.com",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/error@example.com",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, &logger.Noop{})

			res, err := api.DeleteByEmail(tt.email)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, res)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodDelete, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestUsers_GetSentMessages(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.UserSentMessagesRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  []types.UserSentMessage
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful query with email",
			request: types.UserSentMessagesRequest{
				Email:     "test@example.com",
				Limit:     10,
				StartDate: "2025-01-01",
			},
			resBody: []byte(`{
				"messages": [
					{
						"messageId": "123",
						"campaignId": 456,
						"templateId": 789,
						"createdAt": "2025-01-01T00:00:00Z"
					}
				]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/getSentMessages?email=test%40example.com&excludeBlastCampaigns=false&limit=10&startDateTime=2025-01-01",
			expectRes: []types.UserSentMessage{
				{
					MessageId:  "123",
					CampaignId: 456,
					TemplateId: 789,
					CreatedAt:  "2025-01-01T00:00:00Z",
				},
			},
		},
		{
			name: "successful query with userId",
			request: types.UserSentMessagesRequest{
				UserId:      "user123",
				CampaignIds: []int64{1, 2},
			},
			resBody:   []byte(`{"messages": []}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/getSentMessages?campaignIds=1&campaignIds=2&excludeBlastCampaigns=false&userId=user123",
			expectRes: []types.UserSentMessage{},
		},
		{
			name: "server error",
			request: types.UserSentMessagesRequest{
				UserId:      "user456",
				CampaignIds: []int64{1, 2},
			},
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/getSentMessages?campaignIds=1&campaignIds=2&excludeBlastCampaigns=false&userId=user456",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "429",
			request: types.UserSentMessagesRequest{
				UserId:      "user456",
				CampaignIds: []int64{1, 2},
			},
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/getSentMessages?campaignIds=1&campaignIds=2&excludeBlastCampaigns=false&userId=user456",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "network error",
			request: types.UserSentMessagesRequest{
				UserId:      "user456",
				CampaignIds: []int64{1, 2},
			},
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/getSentMessages?campaignIds=1&campaignIds=2&excludeBlastCampaigns=false&userId=user456",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, &logger.Noop{})

			messages, err := api.GetSentMessages(tt.request)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, messages)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestUsers_GetAllFields(t *testing.T) {
	testCases := []struct {
		name       string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  map[string]string
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful fields retrieval",
			resBody: []byte(`{
				"fields": {
					"firstName": "string",
					"age": "number",
					"isActive": "boolean"
				}
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/getFields",
			expectRes: map[string]string{
				"firstName": "string",
				"age":       "number",
				"isActive":  "boolean",
			},
		},
		{
			name:      "empty fields",
			resBody:   []byte(`{"fields": {}}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/getFields",
			expectRes: map[string]string{},
		},
		{
			name:       "server error",
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/getFields",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/getFields",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/getFields",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, nil)

			fields, err := api.GetAllFields()
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, fields)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestUsers_ForgetAndUnforget(t *testing.T) {
	testCases := []struct {
		name       string
		email      string
		userId     string
		isForget   bool
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name:      "forget by email",
			email:     "forget@example.com",
			isForget:  true,
			resBody:   []byte(`{"code": "Success", "msg": "User forgotten"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/forget",
			expectRes: &types.PostResponse{Code: "Success", Message: "User forgotten"},
		},
		{
			name:      "forget by userId",
			userId:    "user123",
			isForget:  true,
			resBody:   []byte(`{"code": "Success", "msg": "User forgotten"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/forget",
			expectRes: &types.PostResponse{Code: "Success", Message: "User forgotten"},
		},
		{
			name:      "unforget by email",
			email:     "unforget@example.com",
			resBody:   []byte(`{"code": "Success", "msg": "User unforgotten"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/unforget",
			expectRes: &types.PostResponse{Code: "Success", Message: "User unforgotten"},
		},
		{
			name:       "server error",
			email:      "error@example.com",
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/unforget",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			email:      "error@example.com",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/unforget",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			email:      "error@example.com",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/unforget",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, nil)

			var res *types.PostResponse
			var err error

			if tt.isForget {
				if tt.email != "" {
					res, err = api.ForgetByEmail(tt.email)
				} else {
					res, err = api.ForgetByUserId(tt.userId)
				}
			} else {
				if tt.email != "" {
					res, err = api.UnForgetByEmail(tt.email)
				} else {
					res, err = api.UnForgetByUserId(tt.userId)
				}
			}

			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, res)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodPost, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())

			// Verify request body
			var sentRequest types.GdprRequest
			err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
			require.NoError(t, err)
			assert.Equal(t, tt.email, sentRequest.Email)
			assert.Equal(t, tt.userId, sentRequest.UserId)
		})
	}
}

func TestUsers_UpdateEmail(t *testing.T) {
	testCases := []struct {
		name       string
		email      string
		userId     string
		newEmail   string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name:      "successful email update",
			email:     "old@example.com",
			userId:    "user123",
			newEmail:  "new@example.com",
			resBody:   []byte(`{"code": "Success", "msg": "Email updated"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/users/updateEmail",
			expectRes: &types.PostResponse{Code: "Success", Message: "Email updated"},
		},
		{
			name:       "update failure",
			email:      "old@example.com",
			userId:     "user123",
			newEmail:   "invalid@",
			resBody:    []byte(`{"code": "Error", "msg": "Invalid email"}`),
			resCode:    400,
			expectUrl:  "https://api.iterable.com/api/users/updateEmail",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "server error",
			email:      "error@example.com",
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/users/updateEmail",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			email:      "error@example.com",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/users/updateEmail",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			email:      "error@example.com",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/users/updateEmail",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewUsersApi(testApiKey, c, nil)

			res, err := api.UpdateEmail(tt.email, tt.userId, tt.newEmail)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, res)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodPost, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())

			var sentRequest types.UserUpdateEmailRequest
			err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
			require.NoError(t, err)
			assert.Equal(t, tt.email, sentRequest.Email)
			assert.Equal(t, tt.userId, sentRequest.UserId)
			assert.Equal(t, tt.newEmail, sentRequest.NewEmail)
		})
	}
}
