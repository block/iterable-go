package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"iterable-go/errors"
	"iterable-go/logger"
	"iterable-go/types"

	"github.com/stretchr/testify/assert"
)

func TestNewCampaignsApi(t *testing.T) {
	client := &http.Client{}
	api := NewCampaignsApi(testApiKey, client, &logger.Noop{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.api)
	assert.Equal(t, testApiKey, api.api.apiKey)
	assert.Equal(t, client, api.api.httpClient)
}

func TestCampaigns_All(t *testing.T) {
	testCases := []struct {
		name       string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  []types.Campaign
		expectErr  bool
		resErrType string
	}{
		{
			name:      "successful response",
			resBody:   []byte(`{"campaigns": [{"id": 123, "name": "Test Campaign"}]}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/campaigns",
			expectRes: []types.Campaign{{Id: 123, Name: "Test Campaign"}},
		},
		{
			name:      "empty response",
			resBody:   []byte(`{"campaigns": []}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/campaigns",
			expectRes: []types.Campaign{},
		},
		{
			name:       "malformed json",
			resBody:    []byte(`{"campaigns": [{"id":`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/campaigns",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name:       "server error",
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/campaigns",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "too many requests",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/campaigns",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/campaigns",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCampaignsApi(testApiKey, c, &logger.Noop{})

			campaigns, err := api.All()
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, campaigns)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestCampaigns_Trigger(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.TriggerCampaignRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "ok",
			request: types.TriggerCampaignRequest{
				CampaignId: 14810308,
				ListIds:    []int64{109581},
			},
			resCode:   200,
			resBody:   []byte(`{"code": "Success", "msg": "Campaign triggered"}`),
			expectUrl: "https://api.iterable.com/api/campaigns/trigger",
			expectRes: &types.PostResponse{
				Code:    "Success",
				Message: "Campaign triggered",
			},
		},
		{
			name: "err",
			request: types.TriggerCampaignRequest{
				CampaignId: 14810308,
				ListIds:    []int64{109581},
			},
			resCode: 400,
			resBody: []byte(`{
				"msg":"Campaign is not a triggered campaign",
				"code":"BadParams","params":null}
			`),
			expectUrl:  "https://api.iterable.com/api/campaigns/trigger",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
			expectRes: &types.PostResponse{
				Code:    "BadParams",
				Message: "Campaign is not a triggered campaign",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCampaignsApi(testApiKey, c, &logger.Noop{})

			res, err := api.Trigger(tt.request)
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
				var sentRequest types.TriggerCampaignRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestCampaigns_ActivateTriggered(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.ActivateTriggeredCampaignRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "ok",
			request: types.ActivateTriggeredCampaignRequest{
				CampaignId: 14810308,
			},
			resCode:   200,
			resBody:   []byte(`{"code": "Success", "msg": "Campaign activated"}`),
			expectUrl: "https://api.iterable.com/api/campaigns/activateTriggered",
			expectRes: &types.PostResponse{
				Code:    "Success",
				Message: "Campaign activated",
			},
		},
		{
			name: "err",
			request: types.ActivateTriggeredCampaignRequest{
				CampaignId: 14810308,
			},
			resCode: 400,
			resBody: []byte(`{
				"msg":"Cannot activate a non-triggered campaign 'test-campaign' (Id: 14,810,308)",
				"code":"BadParams",
				"params":null
			}`),
			expectUrl:  "https://api.iterable.com/api/campaigns/activateTriggered",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
			expectRes: &types.PostResponse{
				Code:    "BadParams",
				Message: "Cannot activate a non-triggered campaign 'test-campaign' (Id: 14,810,308)",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCampaignsApi(testApiKey, c, &logger.Noop{})

			res, err := api.ActivateTriggered(tt.request.CampaignId)
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
				var sentRequest types.ActivateTriggeredCampaignRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestCampaigns_Create(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name        string
		request     types.CreateCampaignRequest
		resBody     []byte
		resCode     int
		resErr      error
		expectUrl   string
		expectCamId int64
		expectErr   bool
		resErrType  string
	}{
		{
			name: "ok",
			request: types.CreateCampaignRequest{
				Name:               "test-campaign",
				ListIds:            []int64{101},
				SuppressionListIds: []int64{102},
				TemplateId:         103,
				SendAt:             types.CampaignSentAtFormat(now),
				SendMode:           types.CampaignSendMode.ProjectTimeZone(),
				StartTimeZone:      "America/New_York",
				DefaultTimeZone:    "America/Los_Angeles",
				DataFields: map[string]any{
					"{{unsubscribeUrl}}":          "example.com/unsub",
					"{{hostedUnsubscribeUrl":      "example.com/hosted",
					"{{unsubscribeMessageTypeUrl": "example.com/unsubType",
				},
			},
			resCode:     200,
			resBody:     []byte(`{"campaignId": 14810308}`),
			expectUrl:   "https://api.iterable.com/api/campaigns/create",
			expectCamId: 14810308,
		},
		{
			name: "err",
			request: types.CreateCampaignRequest{
				Name: "test-campaign",
			},
			resCode: 400,
			resBody: []byte(`{
				"msg":"[/api/campaigns/create] Invalid format: \"\"",
				"code":"BadJsonBody",
				"params":null}
			`),
			expectUrl:   "https://api.iterable.com/api/campaigns/create",
			expectErr:   true,
			resErrType:  errors.TYPE_HTTP_STATUS,
			expectCamId: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCampaignsApi(testApiKey, c, &logger.Noop{})

			camId, err := api.Create(tt.request)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
				assert.Equal(t, int64(0), camId)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCamId, camId)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodPost, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())

			// Verify request body
			if !tt.expectErr {
				var sentRequest types.CreateCampaignRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestCampaigns_Abort(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.AbortCampaignRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "ok",
			request: types.AbortCampaignRequest{
				CampaignId: 14810308,
			},
			resCode:   200,
			resBody:   []byte(`{"code": "Success", "msg": "Aborted campaign. Id: 14810308"}`),
			expectUrl: "https://api.iterable.com/api/campaigns/abort",
			expectRes: &types.PostResponse{
				Code:    "Success",
				Message: "Aborted campaign. Id: 14810308",
			},
		},
		{
			name: "err",
			request: types.AbortCampaignRequest{
				CampaignId: 14810308,
			},
			resCode: 400,
			resBody: []byte(`{
				"msg":"Campaign abort failed. Id: 14810308",
				"code":"GenericError",
				"params":null
			}`),
			expectUrl:  "https://api.iterable.com/api/campaigns/abort",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
			expectRes: &types.PostResponse{
				Code:    "BadParams",
				Message: "Campaign abort failed. Id: 14810308",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCampaignsApi(testApiKey, c, &logger.Noop{})

			res, err := api.Abort(tt.request.CampaignId)
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
				var sentRequest types.AbortCampaignRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}

func TestCampaigns_Cancel(t *testing.T) {
	testCases := []struct {
		name       string
		request    types.CancelCampaignRequest
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.PostResponse
		expectErr  bool
		resErrType string
	}{
		{
			name: "ok",
			request: types.CancelCampaignRequest{
				CampaignId: 14810308,
			},
			resCode:   200,
			resBody:   []byte(`{"code": "Success", "msg": "Canceled campaign. Id: 14810308"}`),
			expectUrl: "https://api.iterable.com/api/campaigns/cancel",
			expectRes: &types.PostResponse{
				Code:    "Success",
				Message: "Canceled campaign. Id: 14810308",
			},
		},
		{
			name: "err",
			request: types.CancelCampaignRequest{
				CampaignId: 14810308,
			},
			resCode: 400,
			resBody: []byte(`{
				"msg":"Could not cancel campaign 'test-campaign' (Id: 14,810,308). It's already scheduled to go out. Please abort the campaign.",
				"code":"BadParams",
				"params":null
			}`),
			expectUrl:  "https://api.iterable.com/api/campaigns/cancel",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
			expectRes: &types.PostResponse{
				Code:    "BadParams",
				Message: "Could not cancel campaign 'test-campaign' (Id: 14,810,308). It's already scheduled to go out. Please abort the campaign.",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCampaignsApi(testApiKey, c, &logger.Noop{})

			res, err := api.Cancel(tt.request.CampaignId)
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
				var sentRequest types.AbortCampaignRequest
				err = json.NewDecoder(tr.req.Body).Decode(&sentRequest)
				require.NoError(t, err)
				assert.Equal(t, tt.request, sentRequest)
			}
		})
	}
}
