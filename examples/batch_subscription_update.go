package main

import (
	"fmt"
	"time"

	iterable_go "iterable-go"
	"iterable-go/batch"
	"iterable-go/types"
)

func batch_subscription_update(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	// Create a batch client with custom configuration
	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchFlushQueueSize(4),              // Process when 4 updates accumulate
		iterable_go.WithBatchFlushInterval(6*time.Second),   // Or process every 6 seconds
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	// Start the subscription update processor
	batchClient.SubscriptionUpdate().Start()

	// Add multiple subscription updates for batch processing
	subscriptionUpdates := []types.UserUpdateSubscriptionsRequest{
		{
			Email:                      "user1@example.com",
			EmailListIds:               []int64{12345, 67890}, // Replace with actual list IDs
			UnsubscribedChannelIds:     []int64{},
			UnsubscribedMessageTypeIds: []int64{},
			SubscribedMessageTypeIds:   []int64{111, 222}, // Replace with actual message type IDs
		},
		{
			Email:                      "user2@example.com",
			EmailListIds:               []int64{12345},
			UnsubscribedChannelIds:     []int64{333}, // Replace with actual channel ID
			UnsubscribedMessageTypeIds: []int64{444}, // Replace with actual message type ID
			SubscribedMessageTypeIds:   []int64{111},
		},
		{
			UserId:                     "user3", // Using UserId instead of email
			EmailListIds:               []int64{67890},
			UnsubscribedChannelIds:     []int64{},
			UnsubscribedMessageTypeIds: []int64{},
			SubscribedMessageTypeIds:   []int64{222},
		},
		{
			Email:                      "user4@example.com",
			EmailListIds:               []int64{12345, 67890},
			UnsubscribedChannelIds:     []int64{333},
			UnsubscribedMessageTypeIds: []int64{},
			SubscribedMessageTypeIds:   []int64{111, 222},
		},
	}

	// Add subscription updates to the batch processor
	for i, update := range subscriptionUpdates {
		message := batch.Message{
			Data:     &update,
			MetaData: fmt.Sprintf("sub-update-%d", i+1), // Optional tracking ID
		}
		batchClient.SubscriptionUpdate().Add(message)

		identifier := update.Email
		if identifier == "" {
			identifier = update.UserId
		}
		fmt.Printf("Added subscription update %d to batch: %s\n", i+1, identifier)
	}

	// Listen for responses in a separate goroutine
	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("Subscription update failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("Subscription updated successfully (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("Waiting for batch processing...")
	time.Sleep(10 * time.Second)

	// Stop the processor
	batchClient.SubscriptionUpdate().Stop()
	close(responseChan)

	fmt.Println("Batch subscription update completed!")
}
