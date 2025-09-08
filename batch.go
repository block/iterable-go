package iterable_go

import (
	"github.com/block/iterable-go/batch"
)

type Batch struct {
	config      batchConfig
	client      *Client
	eventTrack  batch.Processor
	listSub     batch.Processor
	listUnSub   batch.Processor
	subUpdate   batch.Processor
	emailUpdate batch.Processor
	userUpdate  batch.Processor
}

func NewBatch(client *Client, opts ...BatchConfigOption) *Batch {
	bConfig := defaultBatchConfig()
	for _, o := range opts {
		o(&bConfig)
	}

	pConfig := batch.ProcessorConfig{
		FlushQueueSize: bConfig.flushQueueSize,
		FlushInterval:  bConfig.flushInterval,
		MaxRetries:     bConfig.retryTimes,
		Retry:          bConfig.retry,
		SendIndividual: func() bool { return bConfig.sendIndividual },
		MaxBufferSize:  bConfig.bufferSize,
		Async:          func() bool { return bConfig.sendAsync },
		Logger:         bConfig.logger,
	}

	return &Batch{
		config: bConfig,
		eventTrack: batch.NewProcessor(
			batch.NewEventTrackHandler(client.Events(), bConfig.logger),
			bConfig.responseChan,
			pConfig,
		),
		listSub: batch.NewProcessor(
			batch.NewListSubscribeHandler(client.Lists(), bConfig.logger),
			bConfig.responseChan,
			pConfig,
		),
		listUnSub: batch.NewProcessor(
			batch.NewListUnSubscribeBatchHandler(client.Lists(), bConfig.logger),
			bConfig.responseChan,
			pConfig,
		),
		subUpdate: batch.NewProcessor(
			batch.NewSubscriptionUpdateHandler(client.Users(), bConfig.logger),
			bConfig.responseChan,
			pConfig,
		),
		emailUpdate: batch.NewProcessor(
			batch.NewUserEmailUpdateHandler(client.Users(), bConfig.logger),
			bConfig.responseChan,
			pConfig,
		),
		userUpdate: batch.NewProcessor(
			batch.NewUserUpdateHandler(client.Users(), bConfig.logger),
			bConfig.responseChan,
			pConfig,
		),
	}
}

func (b *Batch) EventTrack() batch.Processor {
	return b.eventTrack
}

func (b *Batch) ListSubscribe() batch.Processor {
	return b.listSub
}

func (b *Batch) ListUnSubscribe() batch.Processor {
	return b.listUnSub
}

func (b *Batch) SubscriptionUpdate() batch.Processor {
	return b.subUpdate
}

func (b *Batch) EmailUpdate() batch.Processor {
	return b.emailUpdate
}

func (b *Batch) UserUpdate() batch.Processor {
	return b.userUpdate
}

func (b *Batch) StartAll() {
	for _, p := range b.all() {
		p.Start()
	}
}

func (b *Batch) StopAll() {
	for _, p := range b.all() {
		p.Stop()
	}
}

func (b *Batch) all() []batch.Processor {
	return []batch.Processor{
		b.eventTrack,
		b.listSub,
		b.listUnSub,
		b.subUpdate,
		b.emailUpdate,
		b.userUpdate,
	}
}
