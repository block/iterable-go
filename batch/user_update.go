package batch

import (
	"github.com/block/iterable-go/api"
	iterable_errors "github.com/block/iterable-go/errors"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"
)

type userUpdateBatch struct {
	batch          types.BulkUpdateRequest
	reqByEmailMap  map[string][]Message
	reqByUserIdMap map[string][]Message
}

func newUserUpdateBatch(createNewFields *bool) *userUpdateBatch {
	return &userUpdateBatch{
		batch: types.BulkUpdateRequest{
			Users:           make([]types.BulkUpdateUser, 0),
			CreateNewFields: createNewFields,
		},
		reqByEmailMap:  make(map[string][]Message),
		reqByUserIdMap: make(map[string][]Message),
	}
}

type userUpdateHandler struct {
	Client          *api.Users
	CreateNewFields *bool
	logger          logger.Logger
}

var _ Handler = &userUpdateHandler{}

func NewUserUpdateHandler(
	client *api.Users,
	logger logger.Logger,
) Handler {
	return &userUpdateHandler{
		Client: client,
		logger: logger,
	}
}

func (s *userUpdateHandler) SetCreateNewFields(createNewFields bool) *userUpdateHandler {
	s.CreateNewFields = &createNewFields
	return s
}

func (s *userUpdateHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	batchReq, responses := s.generatePayloads(batch, s.CreateNewFields)

	res, err := s.Client.BulkUpdate(batchReq.batch)
	s.logger.Debugf("BulkUpdateRequest response: %+v, err: %+v", res, err)
	// Check for failures and add to responses
	if err != nil {
		// If there is a validation error, send all messages individually
		if apiErr := err.(*iterable_errors.ApiError); apiErr != nil && apiErr.IterableCode == FieldTypeMismatchErr {
			responses = addReqFailuresToResponses(batchReq.reqByEmailMap, responses, FieldTypeMismatchErr, true)
			responses = addReqFailuresToResponses(batchReq.reqByUserIdMap, responses, FieldTypeMismatchErr, true)
			return responses, nil, true
		}
		return nil, err, shouldRetry(err)
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

	return responses, nil, false
}

func (s *userUpdateHandler) ProcessOne(req Message) Response {
	var res Response
	if subData, ok := req.Data.(*types.BulkUpdateUser); ok {
		// Transform BulkUpdateUser to UserRequest
		_, err := s.Client.UpdateOrCreate(types.UserRequest{
			Email:              subData.Email,
			UserId:             subData.UserId,
			DataFields:         subData.DataFields,
			PreferUserId:       subData.PreferUserId,
			MergeNestedObjects: subData.MergeNestedObjects,
			CreateNewFields:    s.CreateNewFields,
		})
		res = Response{
			OriginalReq: req,
			Error:       err,
			Retry:       shouldRetry(err),
		}
	} else {
		s.logger.Errorf("Invalid data type in UserUpdate batch request")
		res = Response{
			OriginalReq: req,
			Error:       InvalidDataErr,
		}
	}

	return res
}

func (s *userUpdateHandler) generatePayloads(
	messages []Message,
	createNewFields *bool,
) (*userUpdateBatch, []Response) {
	var responses []Response
	newBatch := newUserUpdateBatch(createNewFields)
	for _, req := range messages {
		if subData, ok := req.Data.(*types.BulkUpdateUser); ok {
			if subData.Email != "" {
				addReqToMap(newBatch.reqByEmailMap, req, subData.Email)
			} else {
				addReqToMap(newBatch.reqByUserIdMap, req, subData.UserId)
			}
			newBatch.batch.Users = append(newBatch.batch.Users, *subData)
		} else {
			s.logger.Errorf("Invalid data type in UserUpdate batch request")
			responses = append(responses, Response{
				OriginalReq: req,
				Error:       InvalidDataErr,
			})
		}
	}

	return newBatch, responses
}
