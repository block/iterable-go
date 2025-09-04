package iterable_go

import (
	"net/http"

	"iterable-go/api"
)

type Client struct {
	httpClient *http.Client

	campaigns    *api.Campaigns
	catalog      *api.Catalog
	lists        *api.Lists
	channels     *api.Channels
	users        *api.Users
	events       *api.Events
	messageTypes *api.MessageTypes
}

func NewClient(apiKey string, opts ...ConfigOption) *Client {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	httpClient := &http.Client{}
	httpClient.Transport = cfg.transport
	httpClient.Timeout = cfg.timeout

	return &Client{
		httpClient:   httpClient,
		campaigns:    api.NewCampaignsApi(apiKey, httpClient, cfg.logger),
		catalog:      api.NewCatalogApi(apiKey, httpClient, cfg.logger),
		lists:        api.NewListsApi(apiKey, httpClient, cfg.logger),
		channels:     api.NewChannelsApi(apiKey, httpClient, cfg.logger),
		users:        api.NewUsersApi(apiKey, httpClient, cfg.logger),
		events:       api.NewEventsApi(apiKey, httpClient, cfg.logger),
		messageTypes: api.NewMessageTypesApi(apiKey, httpClient, cfg.logger),
	}
}

func (c *Client) Campaigns() *api.Campaigns {
	return c.campaigns
}

func (c *Client) Catalog() *api.Catalog {
	return c.catalog
}

func (c *Client) Lists() *api.Lists {
	return c.lists
}

func (c *Client) Channels() *api.Channels {
	return c.channels
}

func (c *Client) Users() *api.Users {
	return c.users
}

func (c *Client) Events() *api.Events {
	return c.events
}

func (c *Client) MessageTypes() *api.MessageTypes {
	return c.messageTypes
}
