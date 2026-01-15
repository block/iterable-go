package batch

import (
	"errors"

	iterable_errors "github.com/block/iterable-go/errors"
)

const (
	ErrStrDisallowedEventName        = "Disallowed Event Name"
	ErrStrConflictEmails             = "Email conflicts"
	ErrStrConflictUserIds            = "UserId conflicts"
	ErrStrForgottenEmails            = "Email Forgotten"
	ErrStrForgottenUserIds           = "UserId Forgotten"
	ErrStrInvalidDataEmails          = "Invalid Data"
	ErrStrInvalidDataUserIds         = "Invalid Data"
	ErrStrInvalidEmails              = "Malformed Email"
	ErrStrInvalidUserIds             = "Malformed UserId"
	ErrStrNotFoundEmails             = "Email not found"
	ErrStrNotFoundUserIds            = "UserId not found"
	ErrStrValidEmailFailures         = "Internal Error with Email"
	ErrStrValidUserIdFailures        = "Internal Error with UserId"
	ErrStrInvalidDataType            = "Invalid data type in batch request"
	ErrStrInvalidListId              = "List ID not valid for Iterable project"
	ErrStrInvalidBatchProcessorState = "InvalidBatchProcessorStateErr"
	ErrStrProcessOneNotAllowed       = "ProcessOne is not allowed"
	ErrStrProcessBatchNotAllowed     = "ProcessBatch is not allowed"
	errStrIsDisallowedFromTracking   = "is disallowed from tracking"
)

var (
	ErrInvalidListId              = errors.New(ErrStrInvalidListId)
	ErrConflictEmails             = errors.New(ErrStrConflictEmails)
	ErrConflictUserIds            = errors.New(ErrStrConflictUserIds)
	ErrForgottenEmails            = errors.New(ErrStrForgottenEmails)
	ErrForgottenUserIds           = errors.New(ErrStrForgottenUserIds)
	ErrInvalidDataEmails          = errors.New(ErrStrInvalidDataEmails)
	ErrInvalidDataUserIds         = errors.New(ErrStrInvalidDataUserIds)
	ErrInvalidEmails              = errors.New(ErrStrInvalidEmails)
	ErrInvalidUserIds             = errors.New(ErrStrInvalidUserIds)
	ErrNotFoundEmails             = errors.New(ErrStrNotFoundEmails)
	ErrNotFoundUserIds            = errors.New(ErrStrNotFoundUserIds)
	ErrFieldTypeMismatch          = errors.New(iterable_errors.ITERABLE_FieldTypeMismatchErrStr)
	ErrInvalidDataType            = errors.New(ErrStrInvalidDataType)
	ErrValidEmailFailures         = errors.New(ErrStrValidEmailFailures)
	ErrValidUserIdFailures        = errors.New(ErrStrValidUserIdFailures)
	ErrDisallowedEventName        = errors.New(ErrStrDisallowedEventName)
	ErrInvalidBatchProcessorState = errors.New(ErrStrInvalidBatchProcessorState)
	ErrProcessOneNotAllowed       = errors.New(ErrStrProcessOneNotAllowed)
	ErrProcessBatchNotAllowed     = errors.New(ErrStrProcessBatchNotAllowed)

	// ErrClientValidationApiErr represents a client-side validation error that occurs before sending the request.
	// This error indicates that the request data failed validation checks on the client side.
	// It uses "fake" HttpStatusCode=400 to emphasize that retrying won't fix the issue.
	ErrClientValidationApiErr = &iterable_errors.ApiError{
		Stage:          iterable_errors.STAGE_BEFORE_REQUEST,
		Type:           "data_validation_error",
		HttpStatusCode: 400,
		IterableCode:   "iterable_client_data_validation_error",
	}

	// ErrServerValidationApiErr represents a server-side validation error that occurs after sending the request.
	// This error indicates that the request data failed validation checks on the server side with a 400 status code.
	// "Real" HttpStatusCode might be 200, but the error uses "fake" HttpStatusCode=400
	// to emphasize that retrying won't fix the issue.
	ErrServerValidationApiErr = &iterable_errors.ApiError{
		Stage:          iterable_errors.STAGE_AFTER_REQUEST,
		Type:           "data_validation_error",
		HttpStatusCode: 400,
		IterableCode:   "iterable_server_data_validation_error",
	}

	// ErrClientMustRetryBatchApiErr indicates that the operation must be retried as a Batch request.
	// Usually, this happens when sending single (non-batch) requests has some negative consequences,
	// and we want to force the client of the library to send the same message as part of a Batch operation.
	ErrClientMustRetryBatchApiErr = &iterable_errors.ApiError{
		Stage:          iterable_errors.STAGE_BEFORE_REQUEST,
		Type:           "must_retry",
		HttpStatusCode: 500,
		IterableCode:   "iterable_client_must_retry_batch",
	}

	// ErrClientMustRetryOneApiErr indicates that the operation must be retried as an individual
	// request. Usually, this happens when sending a Batch request has some negative consequences,
	// and we want to force the client of the library to send the same request individually.
	ErrClientMustRetryOneApiErr = &iterable_errors.ApiError{
		Stage:          iterable_errors.STAGE_BEFORE_REQUEST,
		Type:           "must_retry",
		HttpStatusCode: 500,
		IterableCode:   "iterable_client_must_retry_one",
	}
)
