package retry

// Retry provides a standardized interface for implementing retry logic
// with different strategies. It allows operations to be retried, with configurable retry
// policies such as exponential backoff, maximum attempts, and custom delay strategies.
//
// The interface is used throughout the Iterable client for handling transient failures in:
// - API requests that may fail due to network issues or rate limiting
// - Batch processing operations that encounter temporary service unavailability
// - Database or external service calls that may experience intermittent failures
//
// Usage Example:
//
//	retry := retry.NewExponentialRetry(
//	    retry.WithInitialDuration(100*time.Millisecond),
//	    retry.WithLogger(myLogger),
//	)
//
//	err := retry.Do(3, "api-call", func(attempt int) (error, retry.ExitStrategy) {
//	    result, err := apiClient.MakeRequest()
//	    if err != nil {
//	        if isRetriableError(err) {
//	            return err, retry.Continue  // Retry this error
//	        }
//	        return err, retry.StopNow     // Don't retry this error
//	    }
//	    return nil, retry.StopNow         // Success, stop retrying
//	})
//
// The RetriableFn function receives the current attempt number (0-based) and returns
// an error and an ExitStrategy. The ExitStrategy determines whether to continue
// retrying (Continue) or stop immediately (StopNow), regardless of remaining attempts.
//
// NOTE: if attempts is 0, the fn is never called.
type Retry interface {
	Do(attempts int, fnName string, fn RetriableFn) error
}

type RetriableFn func(attempt int) (error, ExitStrategy)

type ExitStrategy bool

var StopNow ExitStrategy = true
var Continue ExitStrategy = false
