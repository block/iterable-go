package batch

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	iterable_errors "iterable-go/errors"
	"iterable-go/types"
)

const (
	testEmail  = "email1@email.com"
	testUserId = "user1"
	testListId = 1

	SuccessfulTestData        = "success"
	IndividualSuccessTestData = "individual_success"
	RetryTestData             = "retry"
	FailTestData              = "fail"
)

type TestBatchHandler struct{}

var _ Handler = &TestBatchHandler{}

func NewTestBatchHandler() *TestBatchHandler {
	return &TestBatchHandler{}
}

func (h *TestBatchHandler) ProcessBatch(batch []Message) ([]Response, error, bool) {
	responses := make([]Response, 0)
	for _, req := range batch {
		if req.Data == SuccessfulTestData {
			responses = append(responses, Response{
				OriginalReq: req,
			})
		} else if req.Data == RetryTestData {
			responses = append(responses, Response{
				OriginalReq: req,
				Error: &iterable_errors.ApiError{
					Body:           []byte("fail"),
					HttpStatusCode: 503,
				},
				Retry: true,
			})
		} else {
			responses = append(responses, Response{
				OriginalReq: req,
				Error: &iterable_errors.ApiError{
					Body:           []byte("fail"),
					HttpStatusCode: 400,
				},
			})
		}
	}
	return responses, nil, false
}

func (h *TestBatchHandler) ProcessOne(message Message) Response {
	var res Response
	if message.Data == IndividualSuccessTestData {
		res = Response{
			OriginalReq: message,
		}
	} else if message.Data == RetryTestData {
		res = Response{
			OriginalReq: message,
			Error: &iterable_errors.ApiError{
				Body:           []byte("fail"),
				HttpStatusCode: 503,
			},
			Retry: true,
		}
	} else {
		res = Response{
			OriginalReq: message,
			Error: &iterable_errors.ApiError{
				Body:           []byte("fail"),
				HttpStatusCode: 400,
			},
		}
	}

	return res
}

type fakeTransport struct {
	setBatchFailCnt      int
	batchFailCnt         int
	setBatchRateLimitCnt int
	batchRateLimitCnt    int
	reqCnt               int
	disallowedEventNames []string
}

func NewFakeTransport(failCnt int, rateLimitCnt int) *fakeTransport {
	return &fakeTransport{
		setBatchFailCnt:      failCnt,
		setBatchRateLimitCnt: rateLimitCnt,
	}
}

func (m *fakeTransport) SetDisallowedEventNames(eventNames []string) {
	m.disallowedEventNames = eventNames
}

func (m *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.reqCnt++
	if m.batchFailCnt < m.setBatchFailCnt {
		m.batchFailCnt++
		res := types.PostResponse{
			Message: "fail",
			Code:    "500",
		}
		resBytes, _ := json.Marshal(res)
		return &http.Response{
			StatusCode: 500,
			Body:       ioutil.NopCloser(bytes.NewReader(resBytes)),
		}, nil
	}
	if m.batchRateLimitCnt < m.setBatchRateLimitCnt {
		m.batchRateLimitCnt++
		res := types.PostResponse{
			Message: "fail",
			Code:    "429",
		}
		resBytes, _ := json.Marshal(res)
		return &http.Response{
			StatusCode: 429,
			Body:       ioutil.NopCloser(bytes.NewReader(resBytes)),
		}, nil
	}

	var res interface{}
	switch req.URL.Path {
	case "/api/events/trackBulk":
		res = types.BulkEventsResponse{
			DisallowedEventNames: m.disallowedEventNames,
			FailedUpdates: types.FailedEventUpdates{
				InvalidEmails:    make([]string, 0),
				InvalidUserIds:   make([]string, 0),
				NotFoundEmails:   make([]string, 0),
				NotFoundUserIds:  make([]string, 0),
				ForgottenEmails:  make([]string, 0),
				ForgottenUserIds: make([]string, 0),
			},
		}
	case "/api/users/bulkUpdate":
		res = types.BulkUpdateResponse{
			SuccessCount: 10,
			FailCount:    0,
		}
	case "/api/users/bulkUpdateSubscriptions":
		res = types.BulkUserUpdateSubscriptionsResponse{
			SuccessCount: 10,
			FailCount:    0,
		}
	case "/api/users/update", "/api/users/updateSubscriptions":
		res = types.PostResponse{
			Message: "success",
			Code:    "200",
		}
	case "/api/lists/subscribe":
		res = types.ListSubscribeResponse{
			SuccessCount: 10,
			FailCount:    0,
		}
	case "/api/lists/unsubscribe":
		res = types.ListUnSubscribeResponse{
			SuccessCount: 10,
			FailCount:    0,
		}
	default:
		res = types.PostResponse{
			Message: "success",
			Code:    "200",
		}
	}

	resBytes, _ := json.Marshal(res)
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader(resBytes)),
	}, nil
}
