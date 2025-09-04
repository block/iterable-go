package api

import (
	"iterable-go/logger"
	"iterable-go/types"
	"net/http"
	"strconv"
	"strings"
)

var (
	PathCampaignTrigger           = "campaigns/trigger"
	PathCampaignActivateTriggered = "campaigns/activateTriggered"
	PathCampaignCreate            = "campaigns/create"
	PathCampaignAbort             = "campaigns/abort"
	PathCampaignCancel            = "campaigns/cancel"
	PathChildCampaigns            = "campaigns/recurring/{campaignId}/childCampaigns"
	PathCampaigns                 = "campaigns"
)

// Campaigns implements a set of /api/campaigns API methods,
// See: https://api.iterable.com/api/docs#campaigns_campaigns
type Campaigns struct {
	api *apiClient
}

func NewCampaignsApi(apiKey string, httpClient *http.Client, logger logger.Logger) *Campaigns {
	return &Campaigns{
		api: newApiClient(apiKey, httpClient, logger),
	}
}

func (c *Campaigns) Trigger(req types.TriggerCampaignRequest) (*types.PostResponse, error) {
	var res types.PostResponse
	return toNilErr(&res, c.api.postJson(
		PathCampaignTrigger,
		req,
		&res,
	))
}

func (c *Campaigns) ActivateTriggered(campaignId int64) (*types.PostResponse, error) {
	req := types.ActivateTriggeredCampaignRequest{
		CampaignId: campaignId,
	}
	var res types.PostResponse
	return toNilErr(&res, c.api.postJson(
		PathCampaignActivateTriggered,
		req,
		&res,
	))
}

func (c *Campaigns) Create(req types.CreateCampaignRequest) (int64, error) {
	var res types.CreateCampaignResponse
	return toNilErr(res.CampaignId, c.api.postJson(
		PathCampaignCreate,
		req,
		&res,
	))
}

func (c *Campaigns) Abort(campaignId int64) (*types.PostResponse, error) {
	req := types.AbortCampaignRequest{CampaignId: campaignId}
	var res types.PostResponse
	return toNilErr(&res, c.api.postJson(
		PathCampaignAbort,
		req,
		&res,
	))
}

func (c *Campaigns) Cancel(campaignId int64) (*types.PostResponse, error) {
	req := types.CancelCampaignRequest{CampaignId: campaignId}
	var res types.PostResponse
	return toNilErr(&res, c.api.postJson(
		PathCampaignCancel,
		req,
		&res,
	))
}

func (c *Campaigns) ChildRecurringCampaigns(campaignId int64) ([]types.Campaign, error) {
	var res types.CampaignsResponse
	id := strconv.FormatInt(campaignId, 10)
	path := strings.Replace(PathChildCampaigns, "{campaignId}", id, 1)

	return toNilErr(res.Campaigns, c.api.getJson(path, &res))
}

func (c *Campaigns) All() ([]types.Campaign, error) {
	var res types.CampaignsResponse
	return toNilErr(res.Campaigns, c.api.getJson(PathCampaigns, &res))
}
