package batch

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/block/iterable-go/api"
	"github.com/block/iterable-go/logger"
	"github.com/block/iterable-go/rate"
	"github.com/block/iterable-go/types"

	"github.com/stretchr/testify/assert"
)

func TestEventTrackBatchHandler_ProcessBatch(t *testing.T) {
	tests := []struct {
		name           string
		failCnt        int
		rateLimitCnt   int
		expectedErr    bool
		expectedRetry  bool
		expectedResLen int
	}{
		{
			name:           "Success",
			failCnt:        0,
			rateLimitCnt:   0,
			expectedErr:    false,
			expectedRetry:  false,
			expectedResLen: 10,
		},
		{
			name:           "Fail",
			failCnt:        1,
			rateLimitCnt:   0,
			expectedErr:    true,
			expectedRetry:  true,
			expectedResLen: 0,
		},
		{
			name:           "RateLimit",
			failCnt:        0,
			rateLimitCnt:   1,
			expectedErr:    true,
			expectedRetry:  true,
			expectedResLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := testEventTrackHandler(tt.failCnt, tt.rateLimitCnt)
			batch := generateEventTrackTestBatchMessages(10)
			res, err, retry := handler.ProcessBatch(batch)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResLen, len(res))
			assert.Equal(t, tt.expectedRetry, retry)
		})
	}
}

func TestEventTrackBatchHandler_ProcessBatch_DisallowedEventNames(t *testing.T) {
	tests := []struct {
		name              string
		batch             []Message
		disallowedEvents  []string
		expectedFailures  int
		expectedSuccesses int
	}{
		{
			name: "Single disallowed event with email",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						Email:     "test1@test.com",
						EventName: "disallowedEvent",
					},
				},
				{
					Data: &types.EventTrackRequest{
						Email:     "test2@test.com",
						EventName: "allowedEvent",
					},
				},
			},
			disallowedEvents:  []string{"disallowedEvent"},
			expectedFailures:  1,
			expectedSuccesses: 1,
		},
		{
			name: "Single disallowed event with userId",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						UserId:    "user1",
						EventName: "disallowedEvent",
					},
				},
				{
					Data: &types.EventTrackRequest{
						UserId:    "user2",
						EventName: "allowedEvent",
					},
				},
			},
			disallowedEvents:  []string{"disallowedEvent"},
			expectedFailures:  1,
			expectedSuccesses: 1,
		},
		{
			name: "Multiple disallowed events mixed identifiers",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						Email:     "test1@test.com",
						EventName: "disallowedEvent1",
					},
				},
				{
					Data: &types.EventTrackRequest{
						UserId:    "user1",
						EventName: "disallowedEvent2",
					},
				},
				{
					Data: &types.EventTrackRequest{
						Email:     "test2@test.com",
						EventName: "allowedEvent",
					},
				},
			},
			disallowedEvents:  []string{"disallowedEvent1", "disallowedEvent2"},
			expectedFailures:  2,
			expectedSuccesses: 1,
		},
		{
			name: "No disallowed events",
			batch: []Message{
				{
					Data: &types.EventTrackRequest{
						Email:     "test1@test.com",
						EventName: "allowedEvent1",
					},
				},
				{
					Data: &types.EventTrackRequest{
						UserId:    "user1",
						EventName: "allowedEvent2",
					},
				},
			},
			disallowedEvents:  []string{},
			expectedFailures:  0,
			expectedSuccesses: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := testEventTrackHandlerWithDisallowedEvents(tt.disallowedEvents)
			res, err, retry := handler.ProcessBatch(tt.batch)

			assert.NoError(t, err)
			assert.False(t, retry)
			assert.Equal(t, len(tt.batch), len(res))

			failures := 0
			successes := 0
			for _, r := range res {
				if r.Error != nil {
					assert.ErrorIs(t, r.Error, DisallowedEventNameErr)
					failures++
				} else {
					successes++
				}
			}

			assert.Equal(t, tt.expectedFailures, failures)
			assert.Equal(t, tt.expectedSuccesses, successes)
		})
	}
}

func TestEventTrackBatchHandler_ProcessBatch_DuplicateEvent(t *testing.T) {
	handler := testEventTrackHandler(0, 0)
	batch := []Message{
		{
			Data: &types.EventTrackRequest{
				EventName: "testEvent",
			},
		},
		{
			Data: &types.EventTrackRequest{
				EventName: "testEvent",
			},
		},
	}
	res, err, retry := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, false, retry)

	successCnt := 0
	failCnt := 0
	for _, r := range res {
		if r.Error == nil {
			successCnt++
		} else {
			failCnt++
		}
	}
	assert.Equal(t, 2, successCnt)
	assert.Equal(t, 0, failCnt)
}

func TestEventTrackBatchHandler_ProcessBatch_InvalidData(t *testing.T) {
	handler := testEventTrackHandler(0, 0)
	batch := []Message{
		{
			Data: "invalid",
		},
	}
	res, err, retry := handler.ProcessBatch(batch)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, false, retry)

	assert.Error(t, res[0].Error)
}

func TestEventTrackBatchHandler_ProcessOne(t *testing.T) {
	handler := testEventTrackHandler(0, 0)
	batch := generateEventTrackTestBatchMessages(10)

	res := make([]Response, 0, len(batch))
	for _, req := range batch {
		res = append(res, handler.ProcessOne(req))
	}

	assert.Equal(t, 10, len(res))
}

func TestEventTrackBatchHandler_ProcessOne_InvalidData(t *testing.T) {
	handler := testEventTrackHandler(0, 0)
	req := Message{
		Data: "invalid",
	}
	res := handler.ProcessOne(req)

	assert.Error(t, res.Error)
}

func generateEventTrackTestBatchMessages(cnt int) []Message {
	var batch []Message
	for i := 0; i < cnt; i++ {
		batch = append(batch, Message{
			Data: &types.EventTrackRequest{
				Email:     "testEmail" + strconv.Itoa(i) + "@test.com",
				EventName: "testEvent" + strconv.Itoa(i),
			},
			MetaData: strconv.Itoa(i),
		})
	}
	return batch
}

func testEventTrackHandler(failCnt int, rateLimitCnt int) *eventTrackHandler {
	httpClient := http.Client{}
	httpClient.Transport = NewFakeTransport(failCnt, rateLimitCnt)
	ev := api.NewEventsApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewEventTrackHandler(ev, &logger.Noop{})
	return handler.(*eventTrackHandler)
}

func testEventTrackHandlerWithDisallowedEvents(disallowedEvents []string) *eventTrackHandler {
	transport := NewFakeTransport(0, 0)
	transport.SetDisallowedEventNames(disallowedEvents)

	httpClient := http.Client{}
	httpClient.Transport = transport
	ev := api.NewEventsApi("test", &httpClient, &logger.Noop{}, &rate.NoopLimiter{})
	handler := NewEventTrackHandler(ev, &logger.Noop{})
	return handler.(*eventTrackHandler)
}
