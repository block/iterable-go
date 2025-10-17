package rate

import "net/http"

// Limiter controls request rates to the Iterable API.
//
// The Limiter interface provides rate limiting functionality to prevent
// exceeding Iterable's API rate limits. Implementations can use different
// strategies such as:
//   - Token bucket algorithm
//   - Fixed window counting
//   - Sliding window counting
//   - Leaky bucket algorithm
//
// Example usage:
//
//	type TokenBucketLimiter struct {
//	    rate  float64
//	    burst int
//	}
//
//	func (l *TokenBucketLimiter) Limit(req *http.Request) {
//	    // Apply rate limiting logic before allowing request
//	}
//
// The Limit method is called before each request to potentially delay
// or throttle the request based on the current rate limiting state.
// This helps maintain good API citizenship and prevents rate limit errors.
type Limiter interface {
	// Limit applies rate limiting to the given request. This method
	// should block if necessary to maintain the desired request rate.
	// The implementation can use the request information (method, path, etc.)
	// to apply different rate limits for different endpoints.
	Limit(req *http.Request)
}
