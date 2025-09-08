package api

import (
	"net/http"
	"strings"

	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"
)

var (
	PathCatalogs            = "catalogs"
	PathCatalogFieldMapping = "catalogs/{catalogName}/fieldMappings"
	PathCatalog             = "catalogs/{catalogName}"
	PathCatalogItem         = "catalogs/{catalogName}/items/{itemId}"
	PathCatalogItems        = "catalogs/{catalogName}/items"
)

// Catalog implements a set of /api/catalogs API methods,
// See: https://api.iterable.com/api/docs#catalogs_listCatalogs
type Catalog struct {
	api *apiClient
}

func NewCatalogApi(apiKey string, httpClient *http.Client, logger logger.Logger) *Catalog {
	return &Catalog{
		api: newApiClient(apiKey, httpClient, logger),
	}
}

func (c *Catalog) All() (*types.Catalogs, error) {
	var response types.Catalogs
	return toNilErr(&response, c.api.getJson(PathCatalogs, &response))
}

func (c *Catalog) FieldMapping(catalogName string) (*types.CatalogFieldMapping, error) {
	var response types.CatalogFieldMapping
	path := strings.Replace(PathCatalogFieldMapping, "{catalogName}", catalogName, 1)
	return toNilErr(&response, c.api.getJson(path, &response))
}
