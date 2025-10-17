package api

import (
	"net/http"

	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/rate"
	"github.com/block/iterable-go/types"
)

const pathMessageTypes = "messageTypes"

// MessageTypes implements a set of /api/messageTypes API methods,
// See: https://api.iterable.com/api/docs#messageTypes_messageTypes
type MessageTypes struct {
	api *apiClient
}

func NewMessageTypesApi(apiKey string, httpClient *http.Client, logger logger.Logger, limiter rate.Limiter) *MessageTypes {
	return &MessageTypes{
		api: newApiClient(apiKey, httpClient, logger, limiter),
	}
}

func (m *MessageTypes) Get() (*types.MessageTypeResponse, error) {
	var res types.MessageTypeResponse
	return toNilErr(&res, m.api.getJson(
		pathMessageTypes, &res,
	))
}
