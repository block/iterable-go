package iterable_go

import (
	"reflect"
	"testing"
	"time"

	"iterable-go/batch"
	"iterable-go/logger"
	"iterable-go/retry"

	"github.com/stretchr/testify/assert"
)

func Test_newBatch(t *testing.T) {
	c := NewClient(apiKey, WithTransport(
		batch.NewFakeTransport(0, 0),
	))
	b := NewBatch(c)
	assert.NotNil(t, b)

	b.EventTrack().Start()

	var m batch.Message
	b.EventTrack().Add(m)
	b.EventTrack().Stop()
}

func Test_newBatch_opts(t *testing.T) {
	c := NewClient(apiKey, WithTransport(
		batch.NewFakeTransport(0, 0),
	))
	l := &logger.Noop{}
	r := retry.NewExponentialRetry()
	resChan := make(chan batch.Response)
	b := NewBatch(c,
		WithBatchFlushQueueSize(101),
		WithBatchFlushInterval(102*time.Millisecond),
		WithBatchBufferSize(103),
		WithBatchRetryTimes(104),
		WithBatchRetry(r),
		WithBatchSendAsync(true),
		WithBatchSendIndividual(true),
		WithBatchLogger(l),
		WithBatchResponseListener(resChan),
	)
	assert.NotNil(t, b)
	assert.EqualValues(t,
		batchConfig{
			flushQueueSize: 101,
			flushInterval:  102 * time.Millisecond,
			bufferSize:     103,
			retryTimes:     104,
			retry:          r,
			sendAsync:      true,
			sendIndividual: true,
			logger:         l,
			responseChan:   resChan,
		},
		b.config,
	)
}

func Test_newBatch_init_all_processors(t *testing.T) {
	c := NewClient(apiKey, WithTransport(
		batch.NewFakeTransport(0, 0),
	))
	b := NewBatch(c)

	bVal := reflect.ValueOf(b).Elem()
	bType := reflect.TypeOf(b).Elem()
	for i := 0; i < bType.NumField(); i++ {
		field := bType.Field(i)
		if field.Type.Kind() == reflect.Interface {
			var batchProc batch.Processor
			batchProcType := reflect.TypeOf(&batchProc).Elem()
			if field.Type.Implements(batchProcType) {
				fieldVal := bVal.FieldByName(field.Name)
				assert.False(t, fieldVal.IsNil(), "%s is nil", field.Name)
			}
		}
	}
}

func Test_Batch_StartAll_StopAll(t *testing.T) {
	b := Batch{
		eventTrack:  &FakeProcessor{},
		listSub:     &FakeProcessor{},
		listUnSub:   &FakeProcessor{},
		subUpdate:   &FakeProcessor{},
		emailUpdate: &FakeProcessor{},
		userUpdate:  &FakeProcessor{},
	}

	for _, p := range b.all() {
		assert.False(t, p.(*FakeProcessor).started)
		assert.False(t, p.(*FakeProcessor).stopped)
	}

	b.StartAll()
	for _, p := range b.all() {
		assert.True(t, p.(*FakeProcessor).started)
		assert.False(t, p.(*FakeProcessor).stopped)
	}

	b.StopAll()
	for _, p := range b.all() {
		assert.True(t, p.(*FakeProcessor).started)
		assert.True(t, p.(*FakeProcessor).stopped)
	}
}

type FakeProcessor struct {
	started bool
	stopped bool
	added   int
}

var _ batch.Processor = &FakeProcessor{}

func (f *FakeProcessor) Start() {
	f.started = true
}

func (f *FakeProcessor) Stop() {
	f.stopped = true
}

func (f *FakeProcessor) Add(_ batch.Message) {
	f.added++
}
