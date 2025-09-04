package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"iterable-go/errors"
	"iterable-go/logger"
	"iterable-go/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListsApi(t *testing.T) {
	client := &http.Client{}
	api := NewListsApi(testApiKey, client, &logger.Noop{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.api)
	assert.Equal(t, testApiKey, api.api.apiKey)
	assert.Equal(t, client, api.api.httpClient)
}

func TestLists_All(t *testing.T) {
	testCases := []struct {
		name       string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  []types.List
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful response",
			resBody: []byte(`{
				"lists": [
					{"id": 1, "name": "List 1", "description": "Test List 1"},
					{"id": 2, "name": "List 2", "description": "Test List 2"}
				]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists",
			expectRes: []types.List{
				{Id: 1, Name: "List 1", Description: "Test List 1"},
				{Id: 2, Name: "List 2", Description: "Test List 2"},
			},
		},
		{
			name:      "empty lists",
			resBody:   []byte(`{"lists": []}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists",
			expectRes: []types.List{},
		},
		{
			name:       "malformed json",
			resBody:    []byte(`{"lists": [{]}`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/lists",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name:       "server error",
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/lists",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "server error",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/lists",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "server error",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/lists",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewListsApi(testApiKey, c, &logger.Noop{})

			lists, err := api.All()
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, lists)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestLists_Delete(t *testing.T) {
	testCases := []struct {
		name       string
		listId     int64
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
			listId:    123,
			resBody:   []byte(`{"code": "Success", "msg": "List deleted"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/123",
			expectRes: &types.PostResponse{Code: "Success", Message: "List deleted"},
		},
		{
			name:       "server error",
			listId:     456,
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/lists/456",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			listId:     456,
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/lists/456",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			listId:     456,
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/lists/456",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
		{
			name:       "list not found",
			listId:     456,
			resBody:    []byte(`{"code": "NotFound", "msg": "List not found"}`),
			resCode:    404,
			expectUrl:  "https://api.iterable.com/api/lists/456",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewListsApi(testApiKey, c, &logger.Noop{})

			res, err := api.Delete(tt.listId)
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

func TestLists_Create(t *testing.T) {
	testCases := []struct {
		name        string
		listName    string
		description string
		resBody     []byte
		resCode     int
		resErr      error
		expectUrl   string
		expectRes   int64
		expectErr   bool
		resErrType  string
	}{
		{
			name:        "successful create",
			listName:    "New List",
			description: "Test Description",
			resBody:     []byte(`{"listId": 789}`),
			resCode:     200,
			expectUrl:   "https://api.iterable.com/api/lists",
			expectRes:   789,
		},
		{
			name:        "server error",
			listName:    "Error List",
			description: "Error Description",
			resBody:     []byte(`{"message": "Internal Server Error"}`),
			resCode:     500,
			expectUrl:   "https://api.iterable.com/api/lists",
			expectErr:   true,
			resErrType:  errors.TYPE_HTTP_STATUS,
		},
		{
			name:        "429",
			listName:    "Error List",
			description: "Error Description",
			resBody:     []byte(`Too Many Requests`),
			resCode:     429,
			expectUrl:   "https://api.iterable.com/api/lists",
			expectErr:   true,
			resErrType:  errors.TYPE_HTTP_STATUS,
		},
		{
			name:        "network error",
			listName:    "Error List",
			description: "Error Description",
			resErr:      assert.AnError,
			resCode:     0,
			expectUrl:   "https://api.iterable.com/api/lists",
			expectErr:   true,
			resErrType:  errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewListsApi(testApiKey, c, &logger.Noop{})

			listId, err := api.Create(tt.listName, tt.description)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, listId)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodPost, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())

			if !tt.expectErr {
				var sentRequest types.ListCreateRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.listName, sentRequest.Name)
				assert.Equal(t, tt.description, sentRequest.Description)
			}
		})
	}
}

func TestLists_Subscribe(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.ListSubscribeRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.ListSubscribeResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful subscribe",
			request: types.ListSubscribeRequest{
				ListId: 123,
				Subscribers: []types.ListSubscriber{
					{Email: "test@example.com"},
				},
			},
			resBody:   []byte(`{"successCount": 1, "failCount": 0}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/subscribe",
			expectRes: &types.ListSubscribeResponse{
				SuccessCount: 1,
				FailCount:    0,
			},
		},
		{
			name: "partial success",
			request: types.ListSubscribeRequest{
				ListId: 123,
				Subscribers: []types.ListSubscriber{
					{Email: "valid@example.com"},
					{Email: "invalid@"},
				},
			},
			resBody: []byte(`{
				"successCount": 1,
				"failCount": 1,
				"failedUpdates": {
					"invalidEmails": ["invalid@"]
				}
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/subscribe",
			expectRes: &types.ListSubscribeResponse{
				SuccessCount: 1,
				FailCount:    1,
				FailedUpdates: types.FailedUpdates{
					InvalidEmails: []string{"invalid@"},
				},
			},
		},
		{
			name: "server error",
			request: types.ListSubscribeRequest{
				ListId: 123,
				Subscribers: []types.ListSubscriber{
					{Email: "test@example.com"},
				},
			},
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/lists/subscribe",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "429",
			request: types.ListSubscribeRequest{
				ListId: 123,
				Subscribers: []types.ListSubscriber{
					{Email: "test@example.com"},
				},
			},
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/lists/subscribe",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "network error",
			request: types.ListSubscribeRequest{
				ListId: 123,
				Subscribers: []types.ListSubscriber{
					{Email: "test@example.com"},
				},
			},
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/lists/subscribe",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewListsApi(testApiKey, c, &logger.Noop{})

			res, err := api.Subscribe(tt.request)
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
			if !tt.expectErr {
				var sentRequest types.ListSubscribeRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestLists_UnSubscribe(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.ListUnSubscribeRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.ListUnSubscribeResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful unsubscribe",
			request: types.ListUnSubscribeRequest{
				ListId:             123,
				Subscribers:        []types.ListUnSubscriber{{Email: "test@example.com"}},
				CampaignId:         456,
				ChannelUnsubscribe: false,
			},
			resBody:   []byte(`{"successCount": 1, "failCount": 0}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/unsubscribe",
			expectRes: &types.ListUnSubscribeResponse{
				SuccessCount: 1,
				FailCount:    0,
			},
		},
		{
			name: "partial failure",
			request: types.ListUnSubscribeRequest{
				ListId:      123,
				Subscribers: []types.ListUnSubscriber{{Email: "invalid@"}},
			},
			resBody: []byte(`{
				"successCount": 0,
				"failCount": 1,
				"failedUpdates": {
					"invalidEmails": ["invalid@"]
				}
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/unsubscribe",
			expectRes: &types.ListUnSubscribeResponse{
				SuccessCount: 0,
				FailCount:    1,
				FailedUpdates: types.FailedUpdates{
					InvalidEmails: []string{"invalid@"},
				},
			},
		},
		{
			name: "server error",
			request: types.ListUnSubscribeRequest{
				ListId:      123,
				Subscribers: []types.ListUnSubscriber{{Email: "test@example.com"}},
				CampaignId:  456,
			},
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/lists/unsubscribe",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "429",
			request: types.ListUnSubscribeRequest{
				ListId:      123,
				Subscribers: []types.ListUnSubscriber{{Email: "test@example.com"}},
				CampaignId:  456,
			},
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/lists/unsubscribe",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "network error",
			request: types.ListUnSubscribeRequest{
				ListId:      123,
				Subscribers: []types.ListUnSubscriber{{Email: "test@example.com"}},
				CampaignId:  456,
			},
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/lists/unsubscribe",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewListsApi(testApiKey, c, &logger.Noop{})

			res, err := api.UnSubscribe(tt.request)
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
			if !tt.expectErr {
				var sentRequest types.ListUnSubscribeRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestLists_Users(t *testing.T) {
	testCases := []struct {
		name       string
		listId     int64
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  string
		expectErr  bool
		resErrType string
	}{
		{
			name:      "successful users retrieval",
			listId:    123,
			resBody:   []byte(`email1@example.com,email2@example.com`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/getUsers?listId=123&preferUserId=false",
			expectRes: "email1@example.com,email2@example.com",
		},
		{
			name:      "empty list",
			listId:    456,
			resBody:   []byte(``),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/getUsers?listId=456&preferUserId=false",
			expectRes: "",
		},
		{
			name:       "server error",
			listId:     456,
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/lists/getUsers?listId=456&preferUserId=false",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			listId:     456,
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/lists/getUsers?listId=456&preferUserId=false",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			listId:     456,
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/lists/getUsers?listId=456&preferUserId=false",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewListsApi(testApiKey, c, &logger.Noop{})

			users, err := api.Users(tt.listId)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, users)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestLists_Size(t *testing.T) {
	testCases := []struct {
		name       string
		listId     int64
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  int64
		expectErr  bool
		resErrType string
	}{
		{
			name:      "successful size retrieval",
			listId:    123,
			resBody:   []byte(`100`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/123/size",
			expectRes: 100,
		},
		{
			name:      "empty list",
			listId:    456,
			resBody:   []byte(`0`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/lists/456/size",
			expectRes: 0,
		},
		{
			name:       "server error",
			listId:     789,
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/lists/789/size",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			listId:     789,
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/lists/789/size",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			listId:     789,
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/lists/789/size",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewListsApi(testApiKey, c, &logger.Noop{})

			size, err := api.Size(tt.listId)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, int64(0), size)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, size)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}
