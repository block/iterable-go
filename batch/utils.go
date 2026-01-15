package batch

import (
	"errors"
	"maps"

	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/types"
)

func addReqToMap(reqMap map[string][]Message, req Message, key string) {
	r := reqMap[key]
	reqMap[key] = append(r, req)
}

func toSuccess(reqMap map[string][]Message) []Response {
	var responses []Response
	for _, reqSlice := range reqMap {
		for _, req := range reqSlice {
			responses = append(responses, Response{
				OriginalReq: req,
			})
		}
	}
	return responses
}

func toFailures(
	messages []Message,
	err error,
	retry bool,
) []Response {
	var responses []Response
	for _, m := range messages {
		responses = append(responses, toFailure(m, err, retry))
	}
	return responses
}

func toFailure(
	m Message,
	err error,
	retry bool,
) Response {
	return Response{
		OriginalReq: m,
		Error:       err,
		Retry:       retry,
	}
}

func parseFailures(failedKeys []string, reqMap map[string][]Message, err error) []Response {
	var responses []Response
	if len(failedKeys) == 0 {
		return responses
	}

	// Make a map of counts for each key
	countMap := make(map[string]int)
	for _, key := range failedKeys {
		countMap[key]++
	}

	for key, errCnt := range countMap {
		if reqSlice, ok := reqMap[key]; ok {
			// Retry individually if the number of errors for the key is not equal to the number of requests sent.
			// That way, we can determine which requests failed.
			retry := len(reqSlice) != errCnt
			for _, req := range reqSlice {
				// Set Retry to true so the batch processor will send the messages individually
				responses = append(responses, Response{
					OriginalReq: req,
					Error:       err,
					Retry:       retry,
				})
			}
			delete(reqMap, key)
		}
	}
	return responses
}

func parseDisallowedEventNames(
	disallowedEventNames []string,
	reqMap map[string][]Message,
) []Response {
	var responses []Response
	if len(disallowedEventNames) == 0 {
		return responses
	}

	// Make a map of disallowed event names for faster lookup
	disallowedEventNameMap := make(map[string]bool)
	for _, name := range disallowedEventNames {
		disallowedEventNameMap[name] = true
	}

	for reqKey, reqSlice := range reqMap {
		var newReqSlice []Message
		for _, req := range reqSlice {
			trackReq, ok := req.Data.(*types.EventTrackRequest)
			if ok {
				_, found := disallowedEventNameMap[trackReq.EventName]
				if found {
					responses = append(responses, Response{
						OriginalReq: req,
						Error: errors.Join(
							ErrDisallowedEventName,
							ErrServerValidationApiErr,
						),
					})
					continue
				}
			}
			newReqSlice = append(newReqSlice, req)
		}
		reqMap[reqKey] = newReqSlice
	}
	return responses
}

// batchCanRetryBatch determines if a batch operation can be retried as a whole based on the error type.
// Returns true for network-related failures (status 0), rate limits (status 429), and server errors (status 500+).
// These are typically transient issues where retrying the entire batch might succeed.
func batchCanRetryBatch(err error) bool {
	var apiErr *iterable_errors.ApiError
	if !errors.As(err, &apiErr) {
		return false
	}

	status := apiErr.HttpStatusCode
	return status == 0 || // Request was not sent or context timed out
		status == 429 || // Deadline exceeded or rate limited
		status >= 500 // Any server error
}

// batchCanRetryOne determines if individual requests from a failed batch should be retried separately.
func batchCanRetryOne(err error) bool {
	var apiErr *iterable_errors.ApiError
	if !errors.As(err, &apiErr) {
		return false
	}

	status := apiErr.HttpStatusCode
	return status == 0 || // Request was not sent or context timed out
		status == 408 || // Client Request Timeout
		status == 409 || // Conflict
		status == 413 || // (batch request) Payload Too Large
		status == 429 || // Deadline exceeded or rate limited
		status >= 500 // Any server error
}

// oneCanRetry determines if a single request operation should be retried.
func oneCanRetry(err error) bool {
	var apiErr *iterable_errors.ApiError
	if !errors.As(err, &apiErr) {
		return false
	}

	status := apiErr.HttpStatusCode
	return status == 0 || // Request was not sent or context timed out
		status == 408 || // Client Request Timeout
		status == 429 || // Deadline exceeded or rate limited
		status >= 500 // Any server error
}

func handleBatchError(
	apiErr *iterable_errors.ApiError,
	cannotRetry []Response,
	messagesSent []Message,
) (ProcessBatchResponse, error) {
	var result []Response
	result = append(result, cannotRetry...)

	if batchCanRetryBatch(apiErr) {
		result = append(result, toFailures(messagesSent, apiErr, true)...)
		return StatusRetryBatch{result, apiErr}, nil
	} else if batchCanRetryOne(apiErr) {
		result = append(result, toFailures(messagesSent, apiErr, true)...)
		return StatusRetryIndividual{result}, nil
	}

	result = append(result, toFailures(messagesSent, apiErr, false)...)
	return StatusCannotRetry{result}, nil
}

func flatValues(inMaps ...map[string][]Message) []Message {
	var res []Message

	for _, m := range inMaps {
		for mValues := range maps.Values(m) {
			res = append(res, mValues...)
		}
	}

	return res
}
