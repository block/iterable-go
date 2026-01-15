package batch

import (
	"errors"

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

func (s *userUpdateHandler) ProcessBatch(batch []Message) (ProcessBatchResponse, error) {
	req, cannotRetry := s.generatePayloads(batch, s.CreateNewFields)

	var result []Response
	result = append(result, cannotRetry...)
	if len(req.batch.Users) == 0 {
		if len(cannotRetry) != 0 {
			return StatusCannotRetry{result}, nil
		}
		return StatusSuccess{result}, nil
	}

	res, err := s.Client.BulkUpdate(req.batch)
	s.logger.Debugf("BulkUpdateRequest response: %+v, err: %+v", res, err)
	// Check for failures and add to responses
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

		return handleBatchError(apiErr, cannotRetry, messages)
	}

	var sentAndFailed []Response
	f := res.FailedUpdates
	// Map email/userId failures back to requests
	sentAndFailed = append(sentAndFailed, parseFailures(f.ConflictEmails, req.reqByEmailMap, ErrConflictEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.ConflictUserIds, req.reqByUserIdMap, ErrConflictUserIds)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.ForgottenEmails, req.reqByEmailMap, ErrForgottenEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.ForgottenUserIds, req.reqByUserIdMap, ErrForgottenUserIds)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidDataEmails, req.reqByEmailMap, ErrInvalidDataEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidDataUserIds, req.reqByUserIdMap, ErrInvalidDataUserIds)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidEmails, req.reqByEmailMap, ErrInvalidEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.InvalidUserIds, req.reqByUserIdMap, ErrInvalidUserIds)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.NotFoundEmails, req.reqByEmailMap, ErrNotFoundEmails)...)
	sentAndFailed = append(sentAndFailed, parseFailures(f.NotFoundUserIds, req.reqByUserIdMap, ErrNotFoundUserIds)...)

	result = append(result, sentAndFailed...)
	result = append(result, toSuccess(req.reqByEmailMap)...)
	result = append(result, toSuccess(req.reqByUserIdMap)...)

	if len(cannotRetry) == 0 && len(sentAndFailed) == 0 {
		return StatusSuccess{result}, nil
	}
	return StatusPartialSuccess{result}, nil
}

func (s *userUpdateHandler) ProcessOne(req Message) Response {
	var res Response
	if data, ok := req.Data.(*types.BulkUpdateUser); ok {
		// Transform BulkUpdateUser to UserRequest
		_, err := s.Client.UpdateOrCreate(types.UserRequest{
			Email:              data.Email,
			UserId:             data.UserId,
			DataFields:         data.DataFields,
			PreferUserId:       data.PreferUserId,
			MergeNestedObjects: data.MergeNestedObjects,
			CreateNewFields:    s.CreateNewFields,
		})
		if err != nil {
			s.logger.Debugf("Failed to process UserUpdate/ProcessOne: %v", err)
		}

		res = Response{
			OriginalReq: req,
			Error:       err,
			Retry:       oneCanRetry(err),
		}
	} else {
		s.logger.Errorf("Invalid data type in UserUpdate/ProcessOne: %t", req.Data)
		res = Response{
			OriginalReq: req,
			Error: errors.Join(
				ErrInvalidDataType,
				ErrClientValidationApiErr,
			),
		}
	}

	return res
}

func (s *userUpdateHandler) generatePayloads(
	messages []Message,
	createNewFields *bool,
) (*userUpdateBatch, []Response) {
	var cannotRetry []Response
	newBatch := newUserUpdateBatch(createNewFields)
	for _, req := range messages {
		if data, ok := req.Data.(*types.BulkUpdateUser); ok {
			if data.Email != "" {
				addReqToMap(newBatch.reqByEmailMap, req, data.Email)
			} else {
				addReqToMap(newBatch.reqByUserIdMap, req, data.UserId)
			}
			newBatch.batch.Users = append(newBatch.batch.Users, *data)
		} else {
			s.logger.Errorf("Invalid data type in UserUpdate/ProcessBatch: %t", req.Data)
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
