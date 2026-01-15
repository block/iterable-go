package batch

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"
)

type listUnSubscribeBatch struct {
	batch          types.ListUnSubscribeRequest
	reqByEmailMap  map[string][]Message
	reqByUserIdMap map[string][]Message
}

func newListUnSubscribeBatch(listId int64) *listUnSubscribeBatch {
	return &listUnSubscribeBatch{
		batch: types.ListUnSubscribeRequest{
			ListId:      listId,
			Subscribers: make([]types.ListUnSubscriber, 0),
		},
		reqByEmailMap:  make(map[string][]Message),
		reqByUserIdMap: make(map[string][]Message),
	}
}

type listUnSubscribeHandler struct {
	client *api.Lists
	logger logger.Logger
}

var _ Handler = &listUnSubscribeHandler{}

func NewListUnSubscribeBatchHandler(
	client *api.Lists,
	logger logger.Logger,
) Handler {
	return &listUnSubscribeHandler{
		client: client,
		logger: logger,
	}
}

func (s *listUnSubscribeHandler) ProcessBatch(batch []Message) (ProcessBatchResponse, error) {
	batchReqs, cannotRetry := s.generatePayloads(batch)

	var result []Response
	result = append(result, cannotRetry...)
	if len(batchReqs) == 0 {
		if len(cannotRetry) != 0 {
			return StatusCannotRetry{result}, nil
		}
		return StatusSuccess{result}, nil
	}

	// Process each batch of ListUnSubscribeRequests
	for _, batchReq := range batchReqs {
		res, err := s.client.UnSubscribe(batchReq.batch)

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

			// Search for the prefix specifically because the API returns "success" as the code,
			// but it isn't a success
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

func (s *listUnSubscribeHandler) ProcessOne(req Message) Response {
	// This is a no-op for the listUnSubscribeHandler, we don't want to send
	// individual requests for ListUnSubscribe since it will use up our rate limit.
	// We can nack the message, it should get retried as a batches.
	err := errors.Join(ErrProcessOneNotAllowed, ErrClientMustRetryBatchApiErr)
	return toFailure(req, err, true)
}

func (s *listUnSubscribeHandler) generatePayloads(
	messages []Message,
) ([]*listUnSubscribeBatch, []Response) {
	var batchReqs []*listUnSubscribeBatch
	var cannotRetry []Response

	batchMap := make(map[int64]*listUnSubscribeBatch)
	for _, req := range messages {
		if subData, ok := req.Data.(*types.ListUnSubscribeRequest); ok {
			newBatch, batchFound := batchMap[subData.ListId]
			if !batchFound {
				newBatch = newListUnSubscribeBatch(subData.ListId)
				batchMap[subData.ListId] = newBatch
				batchReqs = append(batchReqs, newBatch)
			}

			for _, subscriber := range subData.Subscribers {
				if subscriber.Email != "" {
					addReqToMap(newBatch.reqByEmailMap, req, subscriber.Email)
				} else {
					addReqToMap(newBatch.reqByUserIdMap, req, subscriber.UserId)
				}
				newBatch.batch.Subscribers = append(newBatch.batch.Subscribers, subscriber)
			}
		} else {
			s.logger.Errorf("Invalid data type in ListUnSubscribe/ProcessBatch: %t", req.Data)
			m := toFailure(
				req,
				errors.Join(ErrInvalidDataType, ErrClientValidationApiErr),
				false,
			)
			cannotRetry = append(cannotRetry, m)
		}
	}

	return batchReqs, cannotRetry
}
