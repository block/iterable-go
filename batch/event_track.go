package batch

import (
	"errors"
	"strings"

	"github.com/block/iterable-go/logger"

	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/types"
)

type eventTrackBatch struct {
	batch          types.EventTrackBulkRequest
	reqByEmailMap  map[string][]Message
	reqByUserIdMap map[string][]Message
}

func newEventTrackBatch() *eventTrackBatch {
	return &eventTrackBatch{
		batch: types.EventTrackBulkRequest{
			Events: make([]types.EventTrackRequest, 0),
		},
		reqByEmailMap:  make(map[string][]Message),
		reqByUserIdMap: make(map[string][]Message),
	}
}

type eventTrackHandler struct {
	Client *api.Events
	logger logger.Logger
}

var _ Handler = &eventTrackHandler{}

func NewEventTrackHandler(client *api.Events, logger logger.Logger) Handler {
	return &eventTrackHandler{
		Client: client,
		logger: logger,
	}
}

func (s *eventTrackHandler) ProcessBatch(batch []Message) (ProcessBatchResponse, error) {
	req, cannotRetry := s.generatePayloads(batch)

	var result []Response
	result = append(result, cannotRetry...)
	if len(req.batch.Events) == 0 {
		if len(cannotRetry) != 0 {
			return StatusCannotRetry{result}, nil
		}
		return StatusSuccess{result}, nil
	}

	res, err := s.Client.TrackBulk(req.batch)
	s.logger.Debugf("BulkTrackRequest response: %+v, err: %+v", res, err)
	if err != nil {
		var apiErr *iterable_errors.ApiError
		if !errors.As(err, &apiErr) || apiErr == nil {
			return nil, err
		}

		messages := flatValues(req.reqByEmailMap, req.reqByUserIdMap)

		// If there is a validation error, send all messages individually
		if apiErr.IterableCode == iterable_errors.ITERABLE_FieldTypeMismatchErrStr {
			err2 := errors.Join(ErrFieldTypeMismatch, err)
			result = append(result, toFailures(messages, err2, true)...)

			return StatusRetryIndividual{result}, nil
		}

		return handleBatchError(apiErr, result, messages)
	}

	var sentAndFailed []Response
	f := res.FailedUpdates

	sentAndFailed = append(sentAndFailed, parseDisallowedEventNames(res.DisallowedEventNames, req.reqByEmailMap)...)
	sentAndFailed = append(sentAndFailed, parseDisallowedEventNames(res.DisallowedEventNames, req.reqByUserIdMap)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidEmails, req.reqByEmailMap, ErrInvalidEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidUserIds, req.reqByUserIdMap, ErrInvalidUserIds)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.NotFoundEmails, req.reqByEmailMap, ErrNotFoundEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.NotFoundUserIds, req.reqByUserIdMap, ErrNotFoundUserIds)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.ForgottenEmails, req.reqByEmailMap, ErrForgottenEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.ForgottenUserIds, req.reqByUserIdMap, ErrForgottenUserIds)...)

	// Add success to remaining requests
	result = append(result, sentAndFailed...)
	result = append(result, toSuccess(req.reqByEmailMap)...)
	result = append(result, toSuccess(req.reqByUserIdMap)...)

	if len(cannotRetry) == 0 && len(sentAndFailed) == 0 {
		return StatusSuccess{result}, nil
	}
	return StatusPartialSuccess{result}, nil
}

func (s *eventTrackHandler) ProcessOne(req Message) Response {
	var result Response
	if data, ok := req.Data.(*types.EventTrackRequest); ok {
		res, err := s.Client.Track(*data)
		if err != nil {
			s.logger.Debugf("Failed to process EventTrack/ProcessOne: %v", err)
			result = Response{
				OriginalReq: req,
				Error:       err,
				Retry:       oneCanRetry(err),
			}
		} else if isDisallowedEventName(res.Message) {
			s.logger.Debugf("DisallowedEventName in EventTrack/ProcessOne: %s", data.EventName)
			result = Response{
				OriginalReq: req,
				Error: errors.Join(
					ErrDisallowedEventName,
					ErrServerValidationApiErr,
				),
			}
		} else {
			result = Response{
				OriginalReq: req,
			}
		}
	} else {
		s.logger.Errorf("Invalid data type in EventTrack/ProcessOne: %t", req.Data)
		result = Response{
			OriginalReq: req,
			Error: errors.Join(
				ErrInvalidDataType,
				ErrClientValidationApiErr,
			),
		}
	}
	return result
}

func (s *eventTrackHandler) generatePayloads(messages []Message) (*eventTrackBatch, []Response) {
	var cannotRetry []Response
	newBatch := newEventTrackBatch()
	for _, req := range messages {
		if event, ok := req.Data.(*types.EventTrackRequest); ok {
			if event.Email != "" {
				addReqToMap(newBatch.reqByEmailMap, req, event.Email)
			} else {
				addReqToMap(newBatch.reqByUserIdMap, req, event.UserId)
			}
			newBatch.batch.Events = append(newBatch.batch.Events, *event)
		} else {
			s.logger.Errorf("Invalid data type in EventTrack/ProcessBatch: %t", req.Data)
			cannotRetry = append(cannotRetry, Response{
				OriginalReq: req,
				Error:       ErrInvalidDataType,
			})
		}
	}

	return newBatch, cannotRetry
}

func isDisallowedEventName(message string) bool {
	return strings.Contains(message, errStrIsDisallowedFromTracking)
}
