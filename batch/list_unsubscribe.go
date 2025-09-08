package batch

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/retry"
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
	retry  retry.Retry
}

var _ Handler = &listUnSubscribeHandler{}

func NewListUnSubscribeBatchHandler(
	client *api.Lists,
	logger logger.Logger,
) Handler {
	return &listUnSubscribeHandler{
		client: client,
		logger: logger,
		retry: retry.NewExponentialRetry(
			retry.WithInitialDuration(10*time.Millisecond),
			retry.WithLogger(logger),
		),
	}
}

func (s *listUnSubscribeHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	// Convert Message to multiple ListUnSubscribeRequests by listId and create maps of email/userId to request
	batchReqMap, responses := s.generatePayloads(batch)

	// Process each batch of ListUnSubscribeRequests
	for _, batchReq := range batchReqMap {
		var res *types.ListUnSubscribeResponse
		var err error
		err = s.retry.Do(3, "list-unsubscribe", func(attempt int) (error, retry.ExitStrategy) {
			res, err = s.client.UnSubscribe(batchReq.batch)
			s.logger.Debugf("ListSubscribe response: %+v, err: %+v", res, err)
			if shouldRetry(err) {
				return err, retry.Continue
			} else {
				return err, retry.StopNow
			}
		})
		// Check for failures and return error if it is retriable
		// This will cause the entire batch to be retried
		if err != nil {
			if apiErr := err.(*iterable_errors.ApiError); apiErr != nil {
				resp := &types.PostResponse{}
				errU := json.Unmarshal(apiErr.Body, resp)
				if errU == nil {
					// Search for this specifically because the API returns "success" as the code, but it isn't a success
					if strings.Contains(resp.Message, iterable_errors.ITERABLE_InvalidList) {
						responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, InvalidListId, false)
						responses = addReqFailuresToResponses(batchReq.reqByUserIdMap, responses, InvalidListId, false)
					} else {
						responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, apiErr.IterableCode, shouldRetry(err))
						responses = addReqFailuresToResponses(batchReq.reqByUserIdMap, responses, apiErr.IterableCode, shouldRetry(err))
					}
				}
			} else {
				// If there is some non-API error, log it but don't retry
				responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, UnknownError, true)
			}
		} else {
			// Map email/userId failures back to requests
			responses = parseReqFailures(res.FailedUpdates.ConflictEmails, batchReq.reqByEmailMap, responses, ConflictEmailsErr)
			responses = parseReqFailures(res.FailedUpdates.ConflictUserIds, batchReq.reqByUserIdMap, responses, ConflictUserIdsErr)
			responses = parseReqFailures(res.FailedUpdates.ForgottenEmails, batchReq.reqByEmailMap, responses, ForgottenEmailsErr)
			responses = parseReqFailures(res.FailedUpdates.ForgottenUserIds, batchReq.reqByUserIdMap, responses, ForgottenUserIdsErr)
			responses = parseReqFailures(res.FailedUpdates.InvalidDataEmails, batchReq.reqByEmailMap, responses, InvalidDataEmailsErr)
			responses = parseReqFailures(res.FailedUpdates.InvalidDataUserIds, batchReq.reqByUserIdMap, responses, InvalidDataUserIdsErr)
			responses = parseReqFailures(res.FailedUpdates.InvalidEmails, batchReq.reqByEmailMap, responses, InvalidEmailsErr)
			responses = parseReqFailures(res.FailedUpdates.InvalidUserIds, batchReq.reqByUserIdMap, responses, InvalidUserIdsErr)
			responses = parseReqFailures(res.FailedUpdates.NotFoundEmails, batchReq.reqByEmailMap, responses, NotFoundEmailsErr)
			responses = parseReqFailures(res.FailedUpdates.NotFoundUserIds, batchReq.reqByUserIdMap, responses, NotFoundUserIdsErr)

			// Add success to remaining requests
			responses = addReqSuccessToResponses(batchReq.reqByEmailMap, responses)
			responses = addReqSuccessToResponses(batchReq.reqByUserIdMap, responses)
		}
	}

	return responses, nil, false
}

func (s *listUnSubscribeHandler) ProcessOne(req Message) Response {
	// This is a no-op for the listUnSubscribeHandler, we don't want to send
	// individual requests for ListUnSubscribe since it will use up our rate limit
	// We can nack the message and it should get retried as a batches
	return Response{
		OriginalReq: req,
		Error:       NoIndividualRetryError,
		Retry:       true,
	}
}

func (s *listUnSubscribeHandler) generatePayloads(
	messages []Message,
) (map[int64]*listUnSubscribeBatch, []Response) {
	var responses []Response
	batchMap := make(map[int64]*listUnSubscribeBatch)
	for _, req := range messages {
		if subData, ok := req.Data.(*types.ListUnSubscribeRequest); ok {
			newBatch, batchFound := batchMap[subData.ListId]
			if !batchFound {
				newBatch = newListUnSubscribeBatch(subData.ListId)
				batchMap[subData.ListId] = newBatch
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
			s.logger.Errorf("Invalid data type in ListUnSubscribe batch request")
			responses = append(responses, Response{
				OriginalReq: req,
				Error:       InvalidDataErr,
			})
		}
	}

	return batchMap, responses
}
