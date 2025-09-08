package batch

import (
	"errors"

	iterable_errors "github.com/block/iterable-go/errors"
)

const (
	ConflictEmailsErr     = "Email conflicts"
	ConflictUserIdsErr    = "UserId conflicts"
	ForgottenEmailsErr    = "Email Forgotten"
	ForgottenUserIdsErr   = "UserId Forgotten"
	InvalidDataEmailsErr  = "Invalid Data"
	InvalidDataUserIdsErr = InvalidDataEmailsErr
	InvalidEmailsErr      = "Malformed Email"
	InvalidUserIdsErr     = "Malformed UserId"
	NotFoundEmailsErr     = "Email not found"
	NotFoundUserIdsErr    = "UserId not found"
	ValidEmailFailures    = "Internal Error with Email"
	ValidUserIdFailures   = "Internal Error with UserId"

	InvalidDataTypeErrStr = "Invalid data type in batch request"
	InvalidListId         = "List ID not valid for Iterable project"
	UnknownError          = "Subscribe request failed with unknown error"
)

var (
	InvalidDataErr = &iterable_errors.ApiError{
		Stage:          iterable_errors.STAGE_BEFORE_REQUEST,
		Type:           iterable_errors.TYPE_INVALID_DATA,
		HttpStatusCode: 500,
		IterableCode:   InvalidDataTypeErrStr,
	}
)

// Add request to map of requests
func addReqToMap(reqMap map[string][]Message, req Message, key string) {
	r := reqMap[key]
	reqMap[key] = append(r, req)
}

func addReqSuccessToResponses(reqMap map[string][]Message, responses []Response) []Response {
	for _, reqSlice := range reqMap {
		for _, req := range reqSlice {
			responses = append(responses, Response{
				OriginalReq: req,
			})
		}
	}
	return responses
}

func addReqFailuresToResponses(reqMap map[string][]Message, responses []Response, err string, retry bool) []Response {
	respErr := errors.New(err)
	for _, reqSlice := range reqMap {
		for _, req := range reqSlice {
			responses = append(responses, Response{
				OriginalReq: req,
				Error:       respErr,
				Retry:       retry,
			})
		}
	}
	return responses
}

func parseReqFailures(failedKeys []string, reqMap map[string][]Message, responses []Response, err string) []Response {
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
					Error:       errors.New(err),
					Retry:       retry,
				})
			}
			delete(reqMap, key)
		}
	}
	return responses
}

func shouldRetry(err error) bool {
	if err != nil {
		if apiErr, ok := err.(*iterable_errors.ApiError); ok {
			status := apiErr.HttpStatusCode
			if status == 0 || status == 429 || // Deadline exceeded or rate limited
				(status > 200 && (status < 400 || status >= 500)) { // Non 4xx error
				return true
			}
		}
	}
	return false
}
