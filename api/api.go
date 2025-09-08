package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
)

const (
	baseUrl = "https://api.iterable.com/api"
)

type apiClient struct {
	apiKey     string
	httpClient *http.Client
	logger     logger.Logger
}

func newApiClient(
	apiKey string,
	httpClient *http.Client,
	logger logger.Logger,
) *apiClient {
	return &apiClient{
		apiKey:     apiKey,
		httpClient: httpClient,
		logger:     logger,
	}
}

func (c *apiClient) getJson(path string, resData any) *errors.ApiError {
	return c.sendJson(http.MethodGet, path, nil, resData)
}

func (c *apiClient) postJson(path string, reqData, resData any) *errors.ApiError {
	return c.sendJson(http.MethodPost, path, reqData, resData)
}

func (c *apiClient) deleteJson(path string, reqData, resData any) *errors.ApiError {
	return c.sendJson(http.MethodDelete, path, reqData, resData)
}

func (c *apiClient) sendJson(
	httpMethod string,
	path string,
	reqData any,
	resData any,
) *errors.ApiError {
	body, err := c.send(
		httpMethod,
		path,
		reqData,
		"application/json",
		"application/json",
	)
	if err != nil {
		if len(err.Body) > 0 {
			code := iterableErr{}
			err2 := json.Unmarshal(err.Body, &code)
			if err2 == nil {
				err.IterableCode = code.Code
			}
			// Best effort to return some data
			_ = json.Unmarshal(body, resData)
		}
		return err
	}
	jsonErr := json.Unmarshal(body, resData)
	if jsonErr != nil {
		return &errors.ApiError{
			Stage:          errors.STAGE_AFTER_REQUEST,
			Type:           errors.TYPE_JSON_PARSE,
			SourceErr:      jsonErr,
			Body:           body,
			HttpStatusCode: http.StatusOK,
		}
	}
	return nil
}

func (c *apiClient) getText(path string) (string, *errors.ApiError) {
	return c.sendText(http.MethodGet, path, nil)
}

func (c *apiClient) sendText(
	httpMethod string,
	path string,
	reqData any,
) (string, *errors.ApiError) {
	body, err := c.send(
		httpMethod,
		path,
		reqData,
		"text/plain",
		"text/plain",
	)
	if body == nil {
		return "", err
	}
	return string(body), err
}

func (c *apiClient) send(
	httpMethod string,
	path string,
	reqData any,
	contentType string,
	accept string,
) ([]byte, *errors.ApiError) {
	endpoint := baseUrl + "/" + path

	var err error
	var req *http.Request

	if reqData != nil {
		data, jsonErr := json.Marshal(reqData)
		if jsonErr != nil {
			return nil, &errors.ApiError{
				Stage:     errors.STAGE_BEFORE_REQUEST,
				Type:      errors.TYPE_JSON_PARSE,
				SourceErr: jsonErr,
			}
		}
		req, err = http.NewRequest(
			httpMethod, endpoint, bytes.NewBuffer(data),
		)
	} else {
		req, err = http.NewRequest(
			httpMethod, endpoint, nil,
		)
	}

	if err != nil {
		return nil, &errors.ApiError{
			Stage:     errors.STAGE_BEFORE_REQUEST,
			Type:      errors.TYPE_REQUEST_PREP,
			SourceErr: err,
		}
	}

	req.Header.Add("Content-Type", contentType)
	req.Header.Add("Api-Key", c.apiKey)
	req.Header.Set("Accept", accept)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &errors.ApiError{
			Stage:     errors.STAGE_REQUEST,
			Type:      errors.TYPE_IO,
			SourceErr: err,
		}
	}

	if res.StatusCode != http.StatusOK {
		var body []byte
		if res.Body != nil {
			body, _ = io.ReadAll(res.Body)
			defer func() { _ = res.Body.Close() }()
		}
		return body, &errors.ApiError{
			Stage:          errors.STAGE_AFTER_REQUEST,
			Type:           errors.TYPE_HTTP_STATUS,
			Body:           body,
			HttpStatusCode: res.StatusCode,
			SourceErr:      err,
		}
	}

	body, err := io.ReadAll(res.Body)
	defer func() { _ = res.Body.Close() }()
	if err != nil {
		return body, &errors.ApiError{
			Stage:          errors.STAGE_AFTER_REQUEST,
			Type:           errors.TYPE_IO,
			Body:           body,
			HttpStatusCode: res.StatusCode,
			SourceErr:      err,
		}
	}

	return body, nil
}

// toNilErr converts a *errors.ApiError type to be a true nil interface.
// Internally, a Go interface has a Type and Value.
// An interface value is nil only if the V and T are both unset.
// See: https://go.dev/doc/faq#nil_error
func toNilErr[T any](r T, e *errors.ApiError) (T, error) {
	if e != nil {
		return r, e
	}
	return r, nil
}

// notImplemented is used as a temporary solution for methods
// that will be or are being implemented.
//
// Usage:
//
//	func (c *Catalog) Delete(catalogName string) error {
//		return notImplemented(http.MethodDelete, PathCatalog)
//	}
func notImplemented(httpMethod string, endpoint string) error {
	return &errors.ApiError{
		Stage: errors.STAGE_BEFORE_REQUEST,
		Type:  errors.TYPE_NOT_IMPLEMENTED,
		SourceErr: fmt.Errorf(
			"%s %s is not implemented", httpMethod, endpoint,
		),
	}
}

type iterableErr struct {
	Code string `json:"code"`
}
