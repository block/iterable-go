package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/rate"
	"github.com/block/iterable-go/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventsApi(t *testing.T) {
	client := &http.Client{}
	api := NewEventsApi(testApiKey, client, &logger.Noop{}, &rate.NoopLimiter{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.api)
	assert.Equal(t, testApiKey, api.api.apiKey)
	assert.Equal(t, client, api.api.httpClient)
}

func TestEvents_Track(t *testing.T) {
	testCases := []struct {
		name          string
		request       types.EventTrackRequest
		requestString string
		resBody       []byte
		resCode       int
		resErr        error
		expectUrl     string
		expectRes     *types.PostResponse
		expectErr     bool
		resErrType    string
	}{
		{
			name: "successful track event",
			request: types.EventTrackRequest{
				Email:     "test@example.com",
				EventName: "test_event",
				DataFields: map[string]interface{}{
					"field1": "value1",
				},
			},
			requestString: "{\"email\":\"test@example.com\",\"eventName\":\"test_event\",\"createdAt\":0,\"dataFields\":{\"field1\":\"value1\"}}",
			resBody:       []byte(`{"code": "Success", "msg": "Event tracked"}`),
			resCode:       200,
			expectUrl:     "https://api.iterable.com/api/events/track",
			expectRes: &types.PostResponse{
				Code:    "Success",
				Message: "Event tracked",
			},
		},
		{
			name: "malformed response",
			request: types.EventTrackRequest{
				Email:     "test@example.com",
				EventName: "test_event",
			},
			resBody:    []byte(`{malformed`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/events/track",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name: "server error",
			request: types.EventTrackRequest{
				Email:     "test@example.com",
				EventName: "test_event",
			},
			resBody:    []byte(`{"code": "Error", "msg": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/events/track",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "network error",
			request: types.EventTrackRequest{
				Email:     "test@example.com",
				EventName: "test_event",
			},
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/events/track",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.requestString != "" {
				// Check that it writes the right json
				requestBody, _ := json.Marshal(tt.request)
				requestString := string(requestBody)
				assert.Equal(t, tt.requestString, requestString)
			}

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewEventsApi(testApiKey, c, &logger.Noop{}, &rate.NoopLimiter{})

			res, err := api.Track(tt.request)
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

			// Verify request body
			if !tt.expectErr {
				var sentRequest types.EventTrackRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestEvents_TrackBulk(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.EventTrackBulkRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.BulkEventsResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful bulk track events",
			request: types.EventTrackBulkRequest{
				Events: []types.EventTrackRequest{
					{
						Email:     "test1@example.com",
						EventName: "test_event_1",
					},
					{
						Email:     "test2@example.com",
						EventName: "test_event_2",
					},
				},
			},
			resBody:   []byte(`{"successCount": 2}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/events/trackBulk",
			expectRes: &types.BulkEventsResponse{
				SuccessCount: 2,
				FailCount:    0,
			},
		},
		{
			name: "partial success bulk track",
			request: types.EventTrackBulkRequest{
				Events: []types.EventTrackRequest{
					{
						Email:     "invalid@",
						EventName: "test_event",
					},
					{
						Email:     "valid@example.com",
						EventName: "test_event",
					},
				},
			},
			resBody: []byte(`{
				"successCount": 1,
				"failCount": 1,
				"failedUpdates": {"invalidEmails": ["invalid@"]}
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/events/trackBulk",
			expectRes: &types.BulkEventsResponse{
				SuccessCount: 1,
				FailCount:    1,
				FailedUpdates: types.FailedEventUpdates{
					InvalidEmails: []string{"invalid@"},
				},
			},
		},
		{
			name: "server error",
			request: types.EventTrackBulkRequest{
				Events: []types.EventTrackRequest{{Email: "test@example.com", EventName: "test"}},
			},
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/events/trackBulk",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "429",
			request: types.EventTrackBulkRequest{
				Events: []types.EventTrackRequest{{Email: "test@example.com", EventName: "test"}},
			},
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/events/trackBulk",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name: "network error",
			request: types.EventTrackBulkRequest{
				Events: []types.EventTrackRequest{{Email: "test@example.com", EventName: "test"}},
			},
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/events/trackBulk",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewEventsApi(testApiKey, c, &logger.Noop{}, &rate.NoopLimiter{})

			res, err := api.TrackBulk(tt.request)
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

			if !tt.expectErr {
				var sentRequest types.EventTrackBulkRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestEvents_GetByEmail(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	testCases := []struct {
		name       string
		email      string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  []types.Event
		expectErr  bool
		resErrType string
	}{
		{
			name:  "successful get events by email",
			email: "test@example.com",
			resBody: []byte(`{
				"events": [
					{
						"email": "test@example.com",
						"eventName": "test_event",
						"createdAt": "2025-01-01T00:00:00Z"
					}
				]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/events/test@example.com",
			expectRes: []types.Event{
				{
					Email:     "test@example.com",
					EventName: "test_event",
					CreatedAt: testTime,
				},
			},
		},
		{
			name:      "empty events response",
			email:     "noevent@example.com",
			resBody:   []byte(`{"events": []}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/events/noevent@example.com",
			expectRes: []types.Event{},
		},
		{
			name:       "server error",
			email:      "error@example.com",
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/events/error@example.com",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			email:      "error@example.com",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/events/error@example.com",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			email:      "error@example.com",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/events/error@example.com",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewEventsApi(testApiKey, c, &logger.Noop{}, &rate.NoopLimiter{})

			events, err := api.GetByEmail(tt.email)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, events)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestEvents_GetByUserId(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	testCases := []struct {
		name       string
		userId     string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  []types.Event
		expectErr  bool
		resErrType string
	}{
		{
			name:   "successful get events by user ID",
			userId: "user123",
			resBody: []byte(`{
				"events": [
					{
						"email": "test@example.com",
						"eventName": "test_event",
						"createdAt": "2025-01-01T00:00:00Z"
					}
				]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/events/byUserId/user123",
			expectRes: []types.Event{
				{
					Email:     "test@example.com",
					EventName: "test_event",
					CreatedAt: testTime,
				},
			},
		},
		{
			name:      "empty events response",
			userId:    "no-events",
			resBody:   []byte(`{"events": []}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/events/byUserId/no-events",
			expectRes: []types.Event{},
		},
		{
			name:       "malformed",
			userId:     "error123",
			resBody:    []byte(`{"events": [`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/events/byUserId/error123",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name:       "server error",
			userId:     "error123",
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/events/byUserId/error123",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "429",
			userId:     "error123",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/events/byUserId/error123",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			userId:     "error123",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/events/byUserId/error123",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewEventsApi(testApiKey, c, &logger.Noop{}, &rate.NoopLimiter{})

			events, err := api.GetByUserId(tt.userId)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, tt.resErrType, apiError.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, events)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}
