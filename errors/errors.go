package errors

import (
	"errors"
	"fmt"
)

const (
	STAGE_BEFORE_REQUEST = "before-request"
	STAGE_REQUEST        = "request"
	STAGE_AFTER_REQUEST  = "after-request"

	TYPE_UNKNOWN         = "unknown"
	TYPE_NOT_IMPLEMENTED = "not-implemented"
	TYPE_JSON_PARSE      = "json"
	TYPE_REQUEST_PREP    = "request-prep"
	TYPE_IO              = "io"
	TYPE_HTTP_STATUS     = "not-ok-http-status"
	TYPE_INVALID_DATA    = "invalid-data"

	ITERABLE_NoUserWithIdExists      = "error.users.noUserWithIdExists"
	ITERABLE_InvalidList             = "error.lists.invalidListId"
	ITERABLE_Success                 = "Success"
	ITERABLE_FieldTypeMismatchErrStr = "RequestFieldsTypesMismatched"
)

type ApiError struct {
	Stage          string
	Type           string
	SourceErr      error
	Body           []byte
	HttpStatusCode int

	IterableCode string
}

var _ error = &ApiError{}

func (e *ApiError) Error() string {
	var err string
	if e.SourceErr != nil {
		err = e.SourceErr.Error()
	} else {
		err = string(e.Body)
	}
	return fmt.Sprintf(
		"http request to Iterable failed during '%s' stage with error type '%s', httpStatus: '%d'; original err: %v",
		e.Stage, e.Type, e.HttpStatusCode, err,
	)
}

// Is method is required by errors.Is() to properly distinguish between
// different types -vs- same pointer to the same type.
// Without it, errors.Is(err, ErrFieldTypeMismatch) returns false:
// ok := errors.Is(errors.Join(&iterable_errors.ApiError{}), &iterable_errors.ApiError{})
// ^ would be false
func (e *ApiError) Is(other error) bool {
	var err *ApiError
	return errors.As(other, &err) && err != nil
}
