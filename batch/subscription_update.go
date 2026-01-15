package batch

import (
	"errors"

	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"
)

type subscriptionUpdateBatch struct {
	batch          types.BulkUserUpdateSubscriptionsRequest
	reqByEmailMap  map[string][]Message
	reqByUserIdMap map[string][]Message
}

func newSubscriptionUpdateBatch() *subscriptionUpdateBatch {
	return &subscriptionUpdateBatch{
		batch: types.BulkUserUpdateSubscriptionsRequest{
			UpdateSubscriptionsRequests: make([]types.UserUpdateSubscriptionsRequest, 0),
		},
		reqByEmailMap:  make(map[string][]Message),
		reqByUserIdMap: make(map[string][]Message),
	}
}

type subscriptionUpdateHandler struct {
	Client *api.Users
	logger logger.Logger
}

var _ Handler = &subscriptionUpdateHandler{}

func NewSubscriptionUpdateHandler(
	client *api.Users,
	logger logger.Logger,
) Handler {
	return &subscriptionUpdateHandler{
		Client: client,
		logger: logger,
	}
}

func (s *subscriptionUpdateHandler) ProcessBatch(batch []Message) (ProcessBatchResponse, error) {
	req, cannotRetry := s.generatePayloads(batch)

	var result []Response
	result = append(result, cannotRetry...)
	if len(req.batch.UpdateSubscriptionsRequests) == 0 {
		if len(cannotRetry) != 0 {
			return StatusCannotRetry{result}, nil
		}
		return StatusSuccess{result}, nil
	}

	res, err := s.Client.BulkUpdateSubscriptions(req.batch)
	s.logger.Debugf("BulkUpdateSubscriptions response: %+v, err: %+v", res, err)
	// Check for failures and return error if it is retriable
	if err != nil {
		var apiErr *iterable_errors.ApiError
		if !errors.As(err, &apiErr) || apiErr == nil {
			return nil, err
		}

		messages := flatValues(req.reqByEmailMap, req.reqByUserIdMap)
		return handleBatchError(apiErr, result, messages)
	}

	var sentAndFailed []Response
	// Map email/userId failures back to requests
	sentAndFailed = append(sentAndFailed, parseFailures(res.InvalidEmails, req.reqByEmailMap, ErrInvalidEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(res.InvalidUserIds, req.reqByUserIdMap, ErrInvalidUserIds)...)
	sentAndFailed = append(sentAndFailed, parseFailures(res.ValidEmailFailures, req.reqByEmailMap, ErrValidEmailFailures)...)
	sentAndFailed = append(sentAndFailed, parseFailures(res.ValidUserIdFailures, req.reqByUserIdMap, ErrValidUserIdFailures)...)

	result = append(result, sentAndFailed...)
	result = append(result, toSuccess(req.reqByEmailMap)...)
	result = append(result, toSuccess(req.reqByUserIdMap)...)

	if len(cannotRetry) == 0 && len(sentAndFailed) == 0 {
		return StatusSuccess{result}, nil
	}
	return StatusPartialSuccess{result}, nil
}

func (s *subscriptionUpdateHandler) ProcessOne(req Message) Response {
	var res Response
	if data, ok := req.Data.(*types.UserUpdateSubscriptionsRequest); ok {
		_, err := s.Client.UpdateSubscriptions(*data)
		if err != nil {
			s.logger.Debugf("Failed to process SubscriptionUpdate/ProcessOne: %v", err)
		}

		res = Response{
			OriginalReq: req,
			Error:       err,
			Retry:       oneCanRetry(err),
		}
	} else {
		s.logger.Errorf("Invalid data type in SubscriptionUpdate/ProcessOne: %t", req.Data)
		res = Response{
			OriginalReq: req,
			Error: errors.Join(
				ErrInvalidDataType,
				ErrClientValidationApiErr,
			),
		}
	}
	s.logger.Debugf("Successfully sent SubscriptionUpdate/ProcessOne")

	return res
}

func (s *subscriptionUpdateHandler) generatePayloads(
	messages []Message,
) (*subscriptionUpdateBatch, []Response) {
	var cannotRetry []Response
	newBatch := newSubscriptionUpdateBatch()
	for _, req := range messages {
		if data, ok := req.Data.(*types.UserUpdateSubscriptionsRequest); ok {
			if data.Email != "" {
				addReqToMap(newBatch.reqByEmailMap, req, data.Email)
			} else {
				addReqToMap(newBatch.reqByUserIdMap, req, data.UserId)
			}
			newBatch.batch.UpdateSubscriptionsRequests = append(
				newBatch.batch.UpdateSubscriptionsRequests, *data,
			)
		} else {
			s.logger.Errorf("Invalid data type in SubscriptionUpdate/ProcessBatch: %t", req.Data)
			m := toFailure(
				req,
				errors.Join(ErrInvalidDataType, ErrClientValidationApiErr),
				false,
			)
			cannotRetry = append(cannotRetry, m)
		}
	}

	return newBatch, cannotRetry
}
