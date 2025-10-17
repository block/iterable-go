package rate

import "net/http"

type NoopLimiter struct {
}

var _ Limiter = &NoopLimiter{}

func (n NoopLimiter) Limit(_ *http.Request) {
}
