package main

import (
	"fmt"
	"time"

	iterable_go "iterable-go"
	"iterable-go/batch"
	"iterable-go/types"
)

func batch_list_unsubscribe(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchFlushQueueSize(3),              // Process when 3 unsubscriptions accumulate
		iterable_go.WithBatchFlushInterval(4*time.Second),   // Or process every 4 seconds
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	// Start the list unsubscribe processor
	batchClient.ListUnSubscribe().Start()

	// Add multiple list unsubscriptions for batch processing
	listId := int64(12345) // Replace with actual list ID

	unsubscriptions := []types.ListUnSubscribeRequest{
		{
			ListId: listId,
			Subscribers: []types.ListUnSubscriber{
				{
					Email:  "user1@example.com",
					UserId: "user1",
				},
			},
		},
		{
			ListId: listId,
			Subscribers: []types.ListUnSubscriber{
				{
					Email:  "user2@example.com",
					UserId: "user2",
				},
			},
		},
		{
			ListId: listId,
			Subscribers: []types.ListUnSubscriber{
				{
					Email:  "user3@example.com",
					UserId: "user3",
				},
			},
		},
	}

	// Add unsubscriptions to the batch processor
	for i, unsubscription := range unsubscriptions {
		message := batch.Message{
			Data:     &unsubscription,
			MetaData: fmt.Sprintf("unsubscription-%d", i+1), // Optional tracking ID
		}
		batchClient.ListUnSubscribe().Add(message)
		fmt.Printf("Added unsubscription %d to batch: %s from list %d\n",
			i+1, unsubscription.Subscribers[0].Email, unsubscription.ListId)
	}

	// Listen for responses in a separate goroutine
	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("List unsubscription failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("List unsubscription successful (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("Waiting for batch processing...")
	time.Sleep(8 * time.Second)

	// Stop the processor
	batchClient.ListUnSubscribe().Stop()
	close(responseChan)

	fmt.Println("Batch list unsubscription completed!")
}
