package batch

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/retry"
	"github.com/block/iterable-go/types"
)

type listSubscribeBatch struct {
	batch          types.ListSubscribeRequest
	reqByEmailMap  map[string][]Message
	reqByUserIdMap map[string][]Message
}

func newListSubscribeBatch(listId int64, updateExistingUsersOnly bool) *listSubscribeBatch {
	return &listSubscribeBatch{
		batch: types.ListSubscribeRequest{
			ListId:                  listId,
			Subscribers:             make([]types.ListSubscriber, 0),
			UpdateExistingUsersOnly: updateExistingUsersOnly,
		},
		reqByEmailMap:  make(map[string][]Message),
		reqByUserIdMap: make(map[string][]Message),
	}
}

type listSubscribeHandler struct {
	client *api.Lists
	logger logger.Logger
	retry  retry.Retry
}

var _ Handler = &listSubscribeHandler{}

func NewListSubscribeHandler(
	client *api.Lists,
	logger logger.Logger,
) Handler {
	return &listSubscribeHandler{
		client: client,
		logger: logger,
	}
}

func (s *listSubscribeHandler) ProcessBatch(batch []Message) (ProcessBatchResponse, error) {
	batchReqs, cannotRetry := s.generatePayloads(batch)

	var result []Response
	result = append(result, cannotRetry...)
	if len(batchReqs) == 0 {
		if len(cannotRetry) != 0 {
			return StatusCannotRetry{result}, nil
		}
		return StatusSuccess{result}, nil
	}

	for _, batchReq := range batchReqs {
		res, err := s.client.Subscribe(batchReq.batch)
		// Check for failures and don't try invalid listIds, but make sure to retry other
		// subscribe requests
		if err != nil {
			messages := flatValues(batchReq.reqByEmailMap, batchReq.reqByUserIdMap)

			var apiErr *iterable_errors.ApiError
			if !errors.As(err, &apiErr) || apiErr == nil {
				result = append(result, toFailures(messages, err, false)...)
				continue
			}

			res2 := types.PostResponse{}
			errU := json.Unmarshal(apiErr.Body, &res2)
			if errU != nil {
				err2 := errors.Join(errU, err)
				// If there's 1 bad message that makes the json invalid,
				// retrying individually should allow us to find the message.
				result = append(result, toFailures(messages, err2, true)...)
				continue
			}

			if strings.Contains(res2.Message, iterable_errors.ITERABLE_InvalidList) {
				err2 := errors.Join(ErrInvalidListId, err)
				result = append(result, toFailures(messages, err2, false)...)
				continue
			}

			// Never return StatusRetryBatch, because one call to ProcessBatch
			// spawns multiple calls to Iterable.
			if batchCanRetryOne(apiErr) {
				result = append(result, toFailures(messages, err, true)...)
				continue
			}

			result = append(result, toFailures(messages, err, false)...)
			continue
		}

		var sentAndFailed []Response
		f := res.FailedUpdates

		// Map email/userId failures back to requests
		sentAndFailed = append(sentAndFailed, parseFailures(f.ConflictEmails, batchReq.reqByEmailMap, ErrConflictEmails)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.ConflictUserIds, batchReq.reqByUserIdMap, ErrConflictUserIds)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.ForgottenEmails, batchReq.reqByEmailMap, ErrForgottenEmails)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.ForgottenUserIds, batchReq.reqByUserIdMap, ErrForgottenUserIds)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidDataEmails, batchReq.reqByEmailMap, ErrInvalidDataEmails)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidDataUserIds, batchReq.reqByUserIdMap, ErrInvalidDataUserIds)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidEmails, batchReq.reqByEmailMap, ErrInvalidEmails)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidUserIds, batchReq.reqByUserIdMap, ErrInvalidUserIds)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.NotFoundEmails, batchReq.reqByEmailMap, ErrNotFoundEmails)...)
		sentAndFailed = append(sentAndFailed, parseFailures(f.NotFoundUserIds, batchReq.reqByUserIdMap, ErrNotFoundUserIds)...)

		// Add success to remaining requests
		result = append(result, sentAndFailed...)
		result = append(result, toSuccess(batchReq.reqByEmailMap)...)
		result = append(result, toSuccess(batchReq.reqByUserIdMap)...)
	}

	for _, r := range result {
		if r.Error != nil {
			return StatusPartialSuccess{result}, nil
		}
	}

	return StatusSuccess{result}, nil
}

func (s *listSubscribeHandler) ProcessOne(req Message) Response {
	// This is a no-op for the listSubscribeHandler, we don't want to send
	// individual requests for ListSubscribe since it will use up our rate limit
	// We can nack the message, it should get retried as a batch request.
	err := errors.Join(ErrProcessOneNotAllowed, ErrClientMustRetryBatchApiErr)
	return toFailure(req, err, true)
}

func (s *listSubscribeHandler) generatePayloads(
	messages []Message,
) ([]*listSubscribeBatch, []Response) {
	var batchReqs []*listSubscribeBatch
	var cannotRetry []Response

	batchMap := make(map[int64]*listSubscribeBatch)
	for _, req := range messages {
		if data, ok := req.Data.(*types.ListSubscribeRequest); ok {
			newBatch, batchFound := batchMap[data.ListId]
			if !batchFound {
				// NOTE: The new batch will inherit the UpdateExistingUsersOnly flag,
				// from the first request.
				// This could be a parameter when initializing the batch processor instead
				newBatch = newListSubscribeBatch(data.ListId, data.UpdateExistingUsersOnly)
				batchMap[data.ListId] = newBatch
				batchReqs = append(batchReqs, newBatch)
			}

			for _, subscriber := range data.Subscribers {
				if subscriber.Email != "" {
					addReqToMap(newBatch.reqByEmailMap, req, subscriber.Email)
				} else {
					addReqToMap(newBatch.reqByUserIdMap, req, subscriber.UserId)
				}
				newBatch.batch.Subscribers = append(newBatch.batch.Subscribers, subscriber)
			}
		} else {
			s.logger.Errorf("Invalid data type in ListSubscribe/ProcessBatch: %t", req.Data)
			cannotRetry = append(cannotRetry, Response{
				OriginalReq: req,
				Error:       ErrInvalidDataType,
			})
		}
	}

	return batchReqs, cannotRetry
}
