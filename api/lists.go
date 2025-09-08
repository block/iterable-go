package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"
)

var (
	pathLists            = "lists"
	pathList             = "lists/{listId}"
	pathListsSubscribe   = "lists/subscribe"
	pathListsUnSubscribe = "lists/unsubscribe"
	pathListsGetUsers    = "lists/getUsers?listId={listId}&preferUserId=false"
	pathListSize         = "lists/{listId}/size"
)

// Lists implements a set of /api/events API methods,
// See: https://api.iterable.com/api/docs#lists_getLists
type Lists struct {
	api *apiClient
}

func NewListsApi(apiKey string, httpClient *http.Client, logger logger.Logger) *Lists {
	return &Lists{
		api: newApiClient(apiKey, httpClient, logger),
	}
}

func (c *Lists) All() ([]types.List, error) {
	var response types.ListsResponse
	return toNilErr(response.Lists, c.api.getJson(pathLists, &response))
}

func (c *Lists) Delete(listId int64) (*types.PostResponse, error) {
	var res types.PostResponse
	return toNilErr(&res, c.api.deleteJson(
		strings.Replace(pathList, "{listId}", fmt.Sprint(listId), 1),
		nil,
		&res,
	))
}

func (c *Lists) Create(name string, description string) (int64, error) {
	var res types.ListCreateResponse
	return toNilErr(res.ListId, c.api.postJson(
		pathLists,
		types.ListCreateRequest{
			Name:        name,
			Description: description,
		},
		&res,
	))
}

func (c *Lists) Subscribe(req types.ListSubscribeRequest) (*types.ListSubscribeResponse, error) {
	var res types.ListSubscribeResponse
	return toNilErr(&res, c.api.postJson(
		pathListsSubscribe,
		req,
		&res,
	))
}

func (c *Lists) UnSubscribe(req types.ListUnSubscribeRequest) (*types.ListUnSubscribeResponse, error) {
	var res types.ListUnSubscribeResponse
	return toNilErr(&res, c.api.postJson(
		pathListsUnSubscribe,
		req,
		&res,
	))
}

func (c *Lists) Users(listId int64) (string, error) {
	res, err := c.api.getText(
		strings.Replace(pathListsGetUsers, "{listId}", fmt.Sprint(listId), 1),
	)
	return toNilErr(res, err)
}

func (c *Lists) Size(listId int64) (int64, error) {
	var res int64
	return toNilErr(res, c.api.getJson(
		strings.Replace(pathListSize, "{listId}", fmt.Sprint(listId), 1),
		&res,
	))
}
