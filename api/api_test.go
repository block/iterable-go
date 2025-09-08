package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"

	"github.com/stretchr/testify/assert"
)

const (
	testApiKey = "test-api-key"
)

func Test_getJson(t *testing.T) {
	testCases := []struct {
		name      string
		reqPath   string
		resBody   []byte
		resCode   int
		resErr    error
		expectUrl string
		expectObj types.User
		expectErr bool
	}{
		{
			name:      "200 OK",
			reqPath:   "user1",
			resBody:   []byte(`{"userId":"user-1"}`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/user1",
			expectObj: types.User{UserId: "user-1"},
		},
		{
			name:      "failed to send the request",
			reqPath:   "user2",
			resErr:    fmt.Errorf("test error"),
			expectUrl: "https://api.iterable.com/api/user2",
			expectObj: types.User{},
			expectErr: true,
		},
		{
			name:      "malformed json in response",
			reqPath:   "user3",
			resBody:   []byte(`{"userId":`),
			resCode:   200,
			expectUrl: "https://api.iterable.com/api/user3",
			expectObj: types.User{},
			expectErr: true,
		},
		{
			name:      "400",
			reqPath:   "user400",
			resBody:   []byte(`{"message":"error"}`),
			resCode:   400,
			expectUrl: "https://api.iterable.com/api/user400",
			expectObj: types.User{},
			expectErr: true,
		},
		{
			name:      "500",
			reqPath:   "user?a=b",
			resBody:   []byte(`{"message":"error"}`),
			resCode:   500,
			expectUrl: "https://api.iterable.com/api/user?a=b",
			expectObj: types.User{},
			expectErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := httpClient(tt.resBody, tt.resCode, tt.resErr)
			api := newApiClient(testApiKey, c, &logger.Noop{})

			obj := types.User{}
			err := api.getJson(tt.reqPath, &obj)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.EqualValues(t, tt.expectObj, obj)

			tr, _ := c.Transport.(*testTransport)
			assert.Equal(t, tt.expectUrl, tr.Url())
			assert.Equal(t, http.MethodGet, tr.Method())
			assert.Equal(t, testApiKey, tr.ApiKey())

			cl, _ := tr.res.Body.(*testReader)
			assert.Equal(t, cl.isRead, cl.isClosed)
		})
	}
}

func Test_toNilErr(t *testing.T) {
	var err *errors.ApiError
	var err2 error = err
	if err2 == nil {
		assert.Fail(t, "An interface value is nil only if the V and T are both unset.")
	}

	var err3 error
	_, err3 = toNilErr("ignore", err)
	if err3 != nil {
		assert.Fail(t, "Must be nil")
	}
}

func httpClient(body []byte, code int, err error) *http.Client {
	res := &http.Response{
		StatusCode: code,
		Body:       &testReader{Reader: bytes.NewBuffer(body)},
	}
	return &http.Client{
		Transport: &testTransport{res: res, err: err},
	}
}

type testTransport struct {
	req *http.Request
	res *http.Response
	err error
	url string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.req = req
	return t.res, t.err
}

func (t *testTransport) Method() string {
	return t.req.Method
}

func (t *testTransport) Url() string {
	return t.req.URL.String()
}

func (t *testTransport) ApiKey() string {
	return t.req.Header.Get("Api-Key")
}

type testReader struct {
	isClosed bool
	isRead   bool
	io.Reader
}

func (c *testReader) Close() error {
	c.isClosed = true
	return nil
}

func (c *testReader) Read(p []byte) (n int, err error) {
	c.isRead = true
	return c.Reader.Read(p)
}
