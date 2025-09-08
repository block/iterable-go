package batch

import (
	"github.com/block/iterable-go/api"
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

func (s *subscriptionUpdateHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	batchReq, responses := s.generatePayloads(batch)

	res, err := s.Client.BulkUpdateSubscriptions(batchReq.batch)
	s.logger.Debugf("BulkUpdateSubscriptions response: %+v, err: %+v", res, err)
	// Check for failures and return error if it is retriable
	if err != nil {
		return nil, err, shouldRetry(err)
	} else {
		// Map email/userId failures back to requests
		responses = parseReqFailures(res.InvalidEmails, batchReq.reqByEmailMap, responses, InvalidEmailsErr)
		responses = parseReqFailures(res.InvalidUserIds, batchReq.reqByUserIdMap, responses, InvalidUserIdsErr)
		responses = parseReqFailures(res.ValidEmailFailures, batchReq.reqByEmailMap, responses, ValidEmailFailures)
		responses = parseReqFailures(res.ValidUserIdFailures, batchReq.reqByUserIdMap, responses, ValidUserIdFailures)

		// Add success to remaining requests
		responses = addReqSuccessToResponses(batchReq.reqByEmailMap, responses)
		responses = addReqSuccessToResponses(batchReq.reqByUserIdMap, responses)
	}

	return responses, nil, false
}

func (s *subscriptionUpdateHandler) ProcessOne(req Message) Response {
	var res Response
	if subData, ok := req.Data.(*types.UserUpdateSubscriptionsRequest); ok {
		_, err := s.Client.UpdateSubscriptions(*subData)
		res = Response{
			OriginalReq: req,
			Error:       err,
			Retry:       shouldRetry(err),
		}
	} else {
		s.logger.Errorf("Invalid data type in SubscriptionUpdate batch request")
		res = Response{
			OriginalReq: req,
			Error:       InvalidDataErr,
		}
	}

	return res
}

func (s *subscriptionUpdateHandler) generatePayloads(
	messages []Message,
) (*subscriptionUpdateBatch, []Response) {
	var responses []Response
	newBatch := newSubscriptionUpdateBatch()
	for _, req := range messages {
		if subData, ok := req.Data.(*types.UserUpdateSubscriptionsRequest); ok {
			if subData.Email != "" {
				addReqToMap(newBatch.reqByEmailMap, req, subData.Email)
			} else {
				addReqToMap(newBatch.reqByUserIdMap, req, subData.UserId)
			}
			newBatch.batch.UpdateSubscriptionsRequests = append(newBatch.batch.UpdateSubscriptionsRequests, *subData)
		} else {
			s.logger.Errorf("Invalid data type in SubscriptionUpdate batch request")
			responses = append(responses, Response{
				OriginalReq: req,
				Error:       InvalidDataErr,
			})
		}
	}

	return newBatch, responses
}
