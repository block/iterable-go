package api

import (
	"net/http"
	"strings"

	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/rate"
	"github.com/block/iterable-go/types"
)

var (
	PathEventsTrack     = "events/track"
	PathEventsTrackBulk = "events/trackBulk"
	PathEventsByEmail   = "events/{email}"
	PathEventsByUserId  = "events/byUserId/{userId}"
)

// Events implements a set of /api/events API methods,
// See: https://api.iterable.com/api/docs#events_embedded_track_click
//
// From Iterable API Docs:
// Events are created asynchronously and processed separately from single event (non-bulk) endpoint.
// To make sure events are tracked in order, send them all to the same endpoint (either bulk or non-bulk).
type Events struct {
	api *apiClient
}

func NewEventsApi(apiKey string, httpClient *http.Client, logger logger.Logger, limiter rate.Limiter) *Events {
	return &Events{
		api: newApiClient(apiKey, httpClient, logger, limiter),
	}
}

func (e *Events) Track(req types.EventTrackRequest) (*types.PostResponse, error) {
	var res types.PostResponse
	return toNilErr(&res, e.api.postJson(PathEventsTrack, &req, &res))
}

func (e *Events) TrackBulk(req types.EventTrackBulkRequest) (*types.BulkEventsResponse, error) {
	var res types.BulkEventsResponse
	return toNilErr(&res, e.api.postJson(PathEventsTrackBulk, &req, &res))
}

func (e *Events) GetByEmail(email string) ([]types.Event, error) {
	var res types.EventsResponse
	return toNilErr(res.Events, e.api.getJson(
		strings.Replace(PathEventsByEmail, "{email}", email, 1),
		&res,
	))
}

func (e *Events) GetByUserId(userId string) ([]types.Event, error) {
	var res types.EventsResponse
	return toNilErr(res.Events, e.api.getJson(
		strings.Replace(PathEventsByUserId, "{userId}", userId, 1),
		&res,
	))
}
