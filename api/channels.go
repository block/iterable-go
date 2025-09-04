package api

import (
	"net/http"

	"iterable-go/logger"
	"iterable-go/types"
)

var (
	PathChannels = "channels"
)

// Channels implements a set of /api/channels API methods,
// See: https://api.iterable.com/api/docs#channels_channels
type Channels struct {
	api *apiClient
}

func NewChannelsApi(apiKey string, httpClient *http.Client, logger logger.Logger) *Channels {
	return &Channels{
		api: newApiClient(apiKey, httpClient, logger),
	}
}

func (c *Channels) Channels() ([]types.Channel, error) {
	var response types.ChannelsResponse
	return toNilErr(response.Channels, c.api.getJson(PathChannels, &response))
}
