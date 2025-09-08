package api

import (
	"net/http"
	"testing"

	"github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogApi(t *testing.T) {
	client := &http.Client{}
	api := NewCatalogApi(testApiKey, client, &logger.Noop{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.api)
	assert.Equal(t, testApiKey, api.api.apiKey)
	assert.Equal(t, client, api.api.httpClient)
}

func TestCatalog_All(t *testing.T) {
	testCases := []struct {
		name       string
		resBody    []byte
		resCode    int
		resErr     error
		expectUrl  string
		expectRes  *types.Catalogs
		expectErr  bool
		resErrType string
	}{
		{
			name: "successful response",
			resBody: []byte(`{
				"catalogNames": ["test-catalog"], "totalCatalogsCount": 100
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/catalogs",
			expectRes: &types.Catalogs{
				CatalogNames:       []string{"test-catalog"},
				TotalCatalogsCount: 100,
			},
		},
		{
			name:      "empty response",
			resBody:   []byte(`{"catalogNames":[]}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/catalogs",
			expectRes: &types.Catalogs{
				CatalogNames: []string{},
			},
		},
		{
			name:       "malformed json",
			resBody:    []byte(`{"catalogNames":`),
			resCode:    200,
			expectUrl:  "https://api.iterable.com/api/catalogs",
			expectErr:  true,
			resErrType: errors.TYPE_JSON_PARSE,
		},
		{
			name:       "server error",
			resBody:    []byte(`{"message": "Internal Server Error"}`),
			resCode:    500,
			expectUrl:  "https://api.iterable.com/api/catalogs",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "server error",
			resBody:    []byte(`Too Many Requests`),
			resCode:    429,
			expectUrl:  "https://api.iterable.com/api/catalogs",
			expectErr:  true,
			resErrType: errors.TYPE_HTTP_STATUS,
		},
		{
			name:       "network error",
			resErr:     assert.AnError,
			resCode:    0,
			expectUrl:  "https://api.iterable.com/api/catalogs",
			expectErr:  true,
			resErrType: errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCatalogApi(testApiKey, c, &logger.Noop{})

			catalogs, err := api.All()
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, catalogs)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}

func TestCatalog_FieldMapping(t *testing.T) {
	testCases := []struct {
		name        string
		catalogName string
		resBody     []byte
		resCode     int
		resErr      error
		expectUrl   string
		expectRes   *types.CatalogFieldMapping
		expectErr   bool
		resErrType  string
	}{
		{
			name:        "successful response",
			catalogName: "test-catalog",
			resBody: []byte(`{
				"definedMappings": {"field1": "a"},
				"undefinedFields": ["field2", "field3"]
			}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/catalogs/test-catalog/fieldMappings",
			expectRes: &types.CatalogFieldMapping{
				DefinedMappings: map[string]any{
					"field1": "a", // keep it at 1 to avoid sorting issues
				},
				UndefinedFields: []string{
					"field2",
					"field3",
				},
			},
		},
		{
			name:        "empty response",
			catalogName: "empty-catalog",
			resBody:     []byte(`{"definedMappings":{}}`),
			resCode:     200,
			expectUrl:   "https://api.iterable.com/api/catalogs/empty-catalog/fieldMappings",
			expectRes: &types.CatalogFieldMapping{
				DefinedMappings: map[string]interface{}{},
			},
		},
		{
			name:        "malformed json",
			catalogName: "bad-catalog",
			resBody:     []byte(`{"mappings": [{]}`),
			resCode:     200,
			expectUrl:   "https://api.iterable.com/api/catalogs/bad-catalog/fieldMappings",
			expectErr:   true,
			resErrType:  errors.TYPE_JSON_PARSE,
		},
		{
			name:        "server error",
			catalogName: "error-catalog",
			resBody:     []byte(`{"message": "Internal Server Error"}`),
			resCode:     500,
			expectUrl:   "https://api.iterable.com/api/catalogs/error-catalog/fieldMappings",
			expectErr:   true,
			resErrType:  errors.TYPE_HTTP_STATUS,
		},
		{
			name:        "429",
			catalogName: "error-catalog",
			resBody:     []byte(`Too Many Requests`),
			resCode:     429,
			expectUrl:   "https://api.iterable.com/api/catalogs/error-catalog/fieldMappings",
			expectErr:   true,
			resErrType:  errors.TYPE_HTTP_STATUS,
		},
		{
			name:        "network-error",
			catalogName: "error-catalog",
			resErr:      assert.AnError,
			resCode:     0,
			expectUrl:   "https://api.iterable.com/api/catalogs/error-catalog/fieldMappings",
			expectErr:   true,
			resErrType:  errors.TYPE_IO,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := NewCatalogApi(testApiKey, c, &logger.Noop{})

			mapping, err := api.FieldMapping(tt.catalogName)
			if tt.expectErr {
				assert.Error(t, err)
				apiError := err.(*errors.ApiError)
				assert.Equal(t, tt.resCode, apiError.HttpStatusCode)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRes, mapping)
			}

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())
		})
	}
}
