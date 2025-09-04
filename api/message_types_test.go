package api

import (
	"net/http"
	"testing"

	"iterable-go/errors"
	"iterable-go/logger"
	"iterable-go/types"

	"github.com/stretchr/testify/assert"
)

func TestNewMessageTypesApi(t *testing.T) {
	client := &http.Client{}
	api := NewMessageTypesApi(testApiKey, client, &logger.Noop{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.api)
	assert.Equal(t, testApiKey, api.api.apiKey)
	assert.Equal(t, client, api.api.httpClient)
}

func TestMessageTypes_Get(t *testing.T) {
	testCases := []struct {
		name       string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.MessageTypeResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful response",
			resBody: []byte(`{
				"messageTypes": [
					{
						"id": 1,
						"name": "Email Newsletter",
						"channelId": 100,
						"subscriptionPolicy": "OptIn"
					},
					{
						"id": 2,
						"name": "Push Notification",
						"channelId": 200,
						"subscriptionPolicy": "OptOut"
					}
				]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/messageTypes",
			expectRes: &types.MessageTypeResponse{
				MessageTypes: []types.MessageType{
					{
						Id:                 1,
						Name:               "Email Newsletter",
						ChannelId:          100,
						SubscriptionPolicy: "OptIn",
					},
					{
						Id:                 2,
						Name:               "Push Notification",
						ChannelId:          200,
						SubscriptionPolicy: "OptOut",
					},
				},
			},
		},
		{
			name: "response with all fields",
			resBody: []byte(`{
				"messageTypes": [{
					"id": 1,
					"createdAt": 1,
					"name": "Complete Message Type",
					"channelId": 100,
					"subscriptionPolicy": "OptIn",
					"rateLimitPerMinute": 10,
					"frequencyCap": {
						"days": 2,
						"messages": 3
					}
				}]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/messageTypes",
			expectRes: &types.MessageTypeResponse{
				MessageTypes: []types.MessageType{
					{
						Id:                 1,
						CreatedAt:          1,
						Name:               "Complete Message Type",
						ChannelId:          100,
						SubscriptionPolicy: "OptIn",
						RateLimitPerMinute: 10,
						FrequencyCap: types.FrequencyCap{
							Days:     2,
							Messages: 3,
						},
					},
				},
			},
		},
		{
			name:      "empty message types",
			resBody:   []byte(`{"messageTypes": []}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/messageTypes",
			expectRes: &types.MessageTypeResponse{
				MessageTypes: []types.MessageType{},
			},
		},
		{
			name:       "malformed json response",
			resBody:    []byte(`{"messageTypes": [{]}`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/messageTypes",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name:       "server error",
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/messageTypes",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/messageTypes",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewMessageTypesApi(testApiKey, c, &logger.Noop{})

			res, err := api.Get()
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
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}
