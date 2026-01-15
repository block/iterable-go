package batch

import (
	"errors"

	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/types"
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
	ErrStrFieldTypeMismatch          = iterable_errors.ITERABLE_FieldTypeMismatchErrStr
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
	ErrApiError                   = &iterable_errors.ApiError{}
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
	ErrFieldTypeMismatch          = &ErrFieldTypeMismatchType{}
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

type ErrFieldTypeMismatchType struct {
	fields types.MismatchedFieldsParams
}

func (e *ErrFieldTypeMismatchType) Error() string {
	return ErrStrFieldTypeMismatch
}

func (e *ErrFieldTypeMismatchType) MismatchedFields() types.MismatchedFieldsParams {
	return e.fields
}

// Is method is required by errors.Is() to properly distinguish between
// different types -vs- same pointer to the same type.
// Without it, errors.Is(err, ErrFieldTypeMismatch) returns false:
// ok := errors.Is(errors.Join(ErrFieldTypeMismatch), ErrFieldTypeMismatch)
// ^ would be false
func (e *ErrFieldTypeMismatchType) Is(other error) bool {
	var err *ErrFieldTypeMismatchType
	return errors.As(other, &err) && err != nil
}

var _ error = &ErrFieldTypeMismatchType{}

func newErrFieldTypeMismatch(fields types.MismatchedFieldsParams) *ErrFieldTypeMismatchType {
	return &ErrFieldTypeMismatchType{fields}
}

// Unwrap returns a slice of errors from a joinError.
// Unfortunately, we cannot use errors.Unwrap(),
// because requires a slightly different interface: `Unwrap() error`
// which differs from what's inside the joinError: `Unwrap() []error`
func Unwrap(err error) []error {
	u, ok := err.(interface {
		Unwrap() []error
	})
	if !ok {
		return nil
	}
	return u.Unwrap()
}
