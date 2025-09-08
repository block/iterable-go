package batch

import (
	"fmt"

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

func (s *userEmailUpdateHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	var responses []Response
	for _, req := range batch {
		responses = append(responses, Response{
			OriginalReq: req,
			Error:       fmt.Errorf("batch processing not allowed for UserEmailUpdate"),
			Retry:       true,
		})
	}
	return responses, nil, false
}

func (s *userEmailUpdateHandler) ProcessOne(req Message) Response {
	var res Response
	if subData, ok := req.Data.(*types.UserUpdateEmailRequest); ok {
		// Transform BulkUpdateUser to UserRequest
		_, err := s.client.UpdateEmail(subData.Email, subData.UserId, subData.NewEmail)
		res = Response{
			OriginalReq: req,
			Error:       err,
			Retry:       shouldRetry(err),
		}
	} else {
		s.logger.Errorf("Invalid data type in UserEmailUpdate batch request")
		res = Response{
			OriginalReq: req,
			Error:       InvalidDataErr,
		}
	}

	return res
}
