package iterable_go

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/block/iterable-go/rate"
	"github.com/stretchr/testify/assert"
)

var (
	apiKey = "__API__KEY__"
)

func Test_newClient(t *testing.T) {
	c := NewClient(apiKey)
	assert.NotNil(t, c)
	assert.Equal(t, 10*time.Second, c.httpClient.Timeout)
	assert.NotNil(t, 10*time.Second, c.httpClient.Transport)
}

func Test_newClient_opts(t *testing.T) {
	tt := &fakeTransport{}
	c := NewClient(
		apiKey,
		WithTimeout(1*time.Second),
		WithTransport(tt),
		WithRateLimiter(&rate.NoopLimiter{}),
	)
	assert.Equal(t, 1*time.Second, c.httpClient.Timeout)
	assert.Equal(t, tt, c.httpClient.Transport)
}

func Test_newClient_init_all_apis(t *testing.T) {
	c := NewClient(apiKey)
	values := reflect.ValueOf(*c)
	types := reflect.TypeOf(*c)
	for i := range values.NumField() {
		field := values.Field(i)
		fieldName := types.Field(i).Name
		if field.IsNil() {
			assert.Fail(t, fmt.Sprintf("%s is not initialized", fieldName))
		}
	}
}

func Test_config_WithTransport(t *testing.T) {
	c := config{}
	WithTransport(&fakeTransport{})(&c)
	assert.NotNil(t, c.transport)
}

func Test_config_WithTimeout(t *testing.T) {
	c := config{}
	WithTimeout(2 * time.Second)(&c)
	assert.Equal(t, 2*time.Second, c.timeout)
}

func Test_config_WithRateLimiter(t *testing.T) {
	c := config{}
	WithRateLimiter(&rate.NoopLimiter{})(&c)
	assert.NotNil(t, c.limiter)
}

type fakeTransport struct {
}

func (f fakeTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, nil
}

var _ http.RoundTripper = &fakeTransport{}
