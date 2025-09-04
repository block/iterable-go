package api

import (
	"net/http"
	"testing"

	"iterable-go/errors"
	"iterable-go/logger"
	"iterable-go/types"

	"github.com/stretchr/testify/assert"
)

func TestNewChannelsApi(t *testing.T) {
	client := &http.Client{}
	api := NewChannelsApi(testApiKey, client, &logger.Noop{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.api)
	assert.Equal(t, testApiKey, api.api.apiKey)
	assert.Equal(t, client, api.api.httpClient)
}

func TestChannels_Channels(t *testing.T) {
	testCases := []struct {
		name       string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  []types.Channel
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful response with multiple channels",
			resBody: []byte(`{
				"channels": [
					{"id": 1, "name": "Email", "channelType": "Marketing", "messageMedium":"Email"},
					{"id": 2, "name": "Push", "channelType": "Marketing", "messageMedium":"Push"}
				]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/channels",
			expectRes: []types.Channel{
				{Id: 1, Name: "Email", ChannelType: "Marketing", MessageMedium: "Email"},
				{Id: 2, Name: "Push", ChannelType: "Marketing", MessageMedium: "Push"},
			},
		},
		{
			name:      "successful response with empty channels",
			resBody:   []byte(`{"channels": []}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/channels",
			expectRes: []types.Channel{},
		},
		{
			name:       "malformed json response",
			resBody:    []byte(`{"channels": [{]}`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/channels",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name:       "server error",
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/channels",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/channels",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/channels",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewChannelsApi(testApiKey, c, &logger.Noop{})

			channels, err := api.Channels()
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, channels)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}
