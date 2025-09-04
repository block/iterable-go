package main

import (
	"fmt"
	"time"

	iterable_go "iterable-go"
	"iterable-go/batch"
	"iterable-go/types"
)

func batch_list_subscribe(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchFlushQueueSize(3),              // Process when 3 subscriptions accumulate
		iterable_go.WithBatchFlushInterval(4*time.Second),   // Or process every 4 seconds
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	batchClient.ListSubscribe().Start()

	// Add multiple list subscriptions for batch processing
	listId := int64(12345) // Replace with actual list ID

	subscriptions := []types.ListSubscribeRequest{
		{
			ListId: listId,
			Subscribers: []types.ListSubscriber{
				{
					Email:  "user1@example.com",
					UserId: "user1",
					DataFields: map[string]interface{}{
						"firstName": "Alice",
						"lastName":  "Smith",
						"source":    "batch_api",
					},
				},
			},
			UpdateExistingUsersOnly: false,
		},
		{
			ListId: listId,
			Subscribers: []types.ListSubscriber{
				{
					Email:  "user2@example.com",
					UserId: "user2",
					DataFields: map[string]interface{}{
						"firstName": "Bob",
						"lastName":  "Johnson",
						"source":    "batch_api",
					},
				},
			},
			UpdateExistingUsersOnly: false,
		},
		{
			ListId: listId,
			Subscribers: []types.ListSubscriber{
				{
					Email:  "user3@example.com",
					UserId: "user3",
					DataFields: map[string]interface{}{
						"firstName": "Carol",
						"lastName":  "Davis",
						"source":    "batch_api",
					},
				},
			},
			UpdateExistingUsersOnly: false,
		},
	}

	// Add subscriptions to the batch processor
	for i, subscription := range subscriptions {
		message := batch.Message{
			Data:     &subscription,
			MetaData: fmt.Sprintf("subscription-%d", i+1), // Optional tracking ID
		}
		batchClient.ListSubscribe().Add(message)
		fmt.Printf("Added subscription %d to batch: %s to list %d\n",
			i+1, subscription.Subscribers[0].Email, subscription.ListId)
	}

	// Listen for responses in a separate goroutine
	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("List subscription failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("List subscription successful (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("Waiting for batch processing...")
	time.Sleep(10 * time.Second)

	// Stop the processor
	batchClient.ListSubscribe().Stop()
	close(responseChan)

	fmt.Println("Batch list subscription completed!")
}
