package batch

import (
	"encoding/json"
	"strings"
	"time"

	"iterable-go/api"
	iterable_errors "iterable-go/errors"
	"iterable-go/logger"
	"iterable-go/retry"
	"iterable-go/types"
)

var (
	NoIndividualRetryError = &iterable_errors.ApiError{
		Stage:          iterable_errors.STAGE_BEFORE_REQUEST,
		Type:           "no_list_subscribe_individual_retry",
		HttpStatusCode: 0,
		IterableCode:   "no_individual_retry",
	}
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
		retry: retry.NewExponentialRetry(
			retry.WithInitialDuration(10*time.Millisecond),
			retry.WithLogger(logger),
		),
	}
}

func (s *listSubscribeHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	batchReqMap, responses := s.generatePayloads(batch)

	for _, batchReq := range batchReqMap {
		var res *types.ListSubscribeResponse
		var err error
		err = s.retry.Do(3, "list-subscribe", func(attempt int) (error, retry.ExitStrategy) {
			res, err = s.client.Subscribe(batchReq.batch)
			s.logger.Debugf("ListSubscribe response: %+v, err: %+v", res, err)
			if shouldRetry(err) {
				return err, retry.Continue
			} else {
				return err, retry.StopNow
			}
		})
		// Check for failures and don't try invalid listIds, but make sure to retry other
		// subscribe requests
		if err != nil {
			if apiErr := err.(*iterable_errors.ApiError); apiErr != nil {
				resp := &types.PostResponse{}
				errU := json.Unmarshal(apiErr.Body, resp)
				if errU == nil {
					if strings.Contains(resp.Message, iterable_errors.ITERABLE_InvalidList) {
						responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, InvalidListId, false)
						responses = addReqFailuresToResponses(batchReq.reqByUserIdMap, responses, InvalidListId, false)
					} else {
						responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, apiErr.IterableCode, shouldRetry(err))
						responses = addReqFailuresToResponses(batchReq.reqByUserIdMap, responses, apiErr.IterableCode, shouldRetry(err))
					}
				}
			} else {
				// If there is some non-API error, log it and retry
				responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, UnknownError, true)
				responses = addReqFailuresToResponses(batchReq.reqByUserIdMap, responses, UnknownError, true)
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

func (s *listSubscribeHandler) ProcessOne(req Message) Response {
	// This is a no-op for the listSubscribeHandler, we don't want to send
	// individual requests for ListSubscribe since it will use up our rate limit
	// We can nack the message and it should get retried as a batchs
	return Response{
		OriginalReq: req,
		Error:       NoIndividualRetryError,
		Retry:       true,
	}
}

func (s *listSubscribeHandler) generatePayloads(
	messages []Message,
) (map[int64]*listSubscribeBatch, []Response) {
	var responses []Response
	batchMap := make(map[int64]*listSubscribeBatch)
	for _, req := range messages {
		if subData, ok := req.Data.(*types.ListSubscribeRequest); ok {
			newBatch, batchFound := batchMap[subData.ListId]
			if !batchFound {
				// The new batch will inherit the UpdateExistingUsersOnly flag from the first request
				// TODO: This could be a parameter when initializing the batch processor instead
				newBatch = newListSubscribeBatch(subData.ListId, subData.UpdateExistingUsersOnly)
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
			s.logger.Errorf("Invalid data type in ListSubscribe batch request")
			responses = append(responses, Response{
				OriginalReq: req,
				Error:       InvalidDataErr,
			})
		}
	}

	return batchMap, responses
}
