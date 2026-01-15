package batch

import (
	"errors"

	"github.com/block/iterable-go/api"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/types"
)

type userEmailUpdateBatch struct {
	batch types.UserUpdateEmailRequest
}

func newUserEmailUpdateBatch() *userEmailUpdateBatch {
	return &userEmailUpdateBatch{
		batch: types.UserUpdateEmailRequest{},
	}
}

type userEmailUpdateHandler struct {
	client *api.Users
	logger logger.Logger
}

var _ Handler = &userEmailUpdateHandler{}

func NewUserEmailUpdateHandler(
	client *api.Users,
	logger logger.Logger,
) Handler {
	return &userEmailUpdateHandler{
		client: client,
		logger: logger,
	}
}

func (s *userEmailUpdateHandler) ProcessBatch(batch []Message) (ProcessBatchResponse, error) {
	var responses []Response
	for _, req := range batch {
		responses = append(responses, Response{
			OriginalReq: req,
			Error: errors.Join(
				ErrProcessBatchNotAllowed,
				ErrClientMustRetryOneApiErr,
			),
			Retry: true,
		})
	}
	return StatusRetryIndividual{responses}, nil
}

func (s *userEmailUpdateHandler) ProcessOne(req Message) Response {
	var res Response
	if data, ok := req.Data.(*types.UserUpdateEmailRequest); ok {
		// Transform BulkUpdateUser to UserRequest
		_, err := s.client.UpdateEmail(data.Email, data.UserId, data.NewEmail)
		if err != nil {
			s.logger.Debugf("Failed to process UserEmailUpdate/ProcessOne: %v", err)
		}

		res = Response{
			OriginalReq: req,
			Error:       err,
			Retry:       oneCanRetry(err),
		}
	} else {
		s.logger.Errorf("Invalid data type in UserEmailUpdate/ProcessOne: %t", req.Data)
		res = Response{
			OriginalReq: req,
			Error: errors.Join(
				ErrInvalidDataType,
				ErrClientValidationApiErr,
			),
		}
	}
	s.logger.Debugf("Successfully sent UserUpdate/ProcessOne")

	return res
}
