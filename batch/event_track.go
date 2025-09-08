package batch

import (
	"strings"

	"github.com/block/iterable-go/logger"

	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/types"
)

const (
	FieldTypeMismatchErr      = "RequestFieldsTypesMismatched"
	DisallowedEventNameErrStr = "Disallowed Event Name"
)

var (
	DisallowedEventNameErr = &iterable_errors.ApiError{
		Stage:          iterable_errors.STAGE_REQUEST,
		Type:           iterable_errors.TYPE_HTTP_STATUS,
		HttpStatusCode: 400,
		IterableCode:   DisallowedEventNameErrStr,
	}
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

func (s *eventTrackHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	batchReq, responses := s.generatePayloads(batch)

	res, err := s.Client.TrackBulk(batchReq.batch)
	s.logger.Debugf("BulkTrackRequest response: %+v, err: %+v", res, err)
	if err != nil {
		// If there is a validation error, send all messages individually
		if apiErr := err.(*iterable_errors.ApiError); apiErr != nil && apiErr.IterableCode == FieldTypeMismatchErr {
			responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, FieldTypeMismatchErr, true)
			responses = addReqFailuresToResponses(batchReq.reqByUserIdMap, responses, FieldTypeMismatchErr, true)
			return responses, nil, true
		}
		return nil, err, shouldRetry(err)
	} else {
		responses = parseDisallowedEventNameFailures(res.DisallowedEventNames, batchReq, responses)
		responses = parseReqFailures(res.FailedUpdates.InvalidEmails, batchReq.reqByEmailMap, responses, InvalidEmailsErr)
		responses = parseReqFailures(res.FailedUpdates.InvalidUserIds, batchReq.reqByUserIdMap, responses, InvalidUserIdsErr)
		responses = parseReqFailures(res.FailedUpdates.NotFoundEmails, batchReq.reqByEmailMap, responses, NotFoundEmailsErr)
		responses = parseReqFailures(res.FailedUpdates.NotFoundUserIds, batchReq.reqByUserIdMap, responses, NotFoundUserIdsErr)
		responses = parseReqFailures(res.FailedUpdates.ForgottenEmails, batchReq.reqByEmailMap, responses, ForgottenEmailsErr)
		responses = parseReqFailures(res.FailedUpdates.ForgottenUserIds, batchReq.reqByUserIdMap, responses, ForgottenUserIdsErr)
		responses = addReqSuccessToResponses(batchReq.reqByEmailMap, responses)
		responses = addReqSuccessToResponses(batchReq.reqByUserIdMap, responses)
	}

	return responses, nil, false
}

func (s *eventTrackHandler) ProcessOne(req Message) Response {
	var res Response
	if subData, ok := req.Data.(*types.EventTrackRequest); ok {
		resp, err := s.Client.Track(*subData)
		if err != nil {
			res = Response{
				OriginalReq: req,
				Error:       err,
				Retry:       shouldRetry(err),
			}
		} else if isDisallowedEventName(resp.Message) {
			res = Response{
				OriginalReq: req,
				Error:       DisallowedEventNameErr,
			}
		} else {
			res = Response{
				OriginalReq: req,
			}
		}
	} else {
		s.logger.Errorf("Invalid data type in EventTrack batch request")
		res = Response{
			OriginalReq: req,
			Error:       InvalidDataErr,
		}
	}

	return res
}

func (s *eventTrackHandler) generatePayloads(messages []Message) (*eventTrackBatch, []Response) {
	var responses []Response
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
			s.logger.Errorf("Invalid data type in EventTrack batch request")
			responses = append(responses, Response{
				OriginalReq: req,
				Error:       InvalidDataErr,
			})
		}
	}

	return newBatch, responses
}

func isDisallowedEventName(message string) bool {
	// Returns true if message includes "is disallowed from tracking"
	return strings.Contains(message, "is disallowed from tracking")
}

func parseDisallowedEventNameFailures(disallowedEventNames []string, batchReq *eventTrackBatch, responses []Response) []Response {
	if len(disallowedEventNames) == 0 {
		return responses
	}

	// Make a map of disallowed event names for faster lookup
	disallowedEventNameMap := make(map[string]bool)
	for _, name := range disallowedEventNames {
		disallowedEventNameMap[name] = true
	}

	responses = parseMapForDisallowedEventName(disallowedEventNameMap, batchReq.reqByEmailMap, responses)
	responses = parseMapForDisallowedEventName(disallowedEventNameMap, batchReq.reqByUserIdMap, responses)
	return responses
}

func parseMapForDisallowedEventName(disallowedEventNameMap map[string]bool, reqMap map[string][]Message, responses []Response) []Response {
	for key, reqSlice := range reqMap {
		newReqSlice := make([]Message, 0, len(reqSlice))
		for _, req := range reqSlice {
			if _, ok := disallowedEventNameMap[req.Data.(*types.EventTrackRequest).EventName]; ok {
				responses = append(responses, Response{
					OriginalReq: req,
					Error:       DisallowedEventNameErr,
				})
			} else {
				newReqSlice = append(newReqSlice, req)
			}
		}
		reqMap[key] = newReqSlice
	}
	return responses
}
