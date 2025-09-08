package main

import (
	"fmt"
	"time"

	iterable_go "github.com/block/iterable-go"
	"github.com/block/iterable-go/batch"
	"github.com/block/iterable-go/types"
)

func batch_start_stop_all(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchFlushQueueSize(2),              // Process when 2 items accumulate
		iterable_go.WithBatchFlushInterval(3*time.Second),   // Or process every 3 seconds
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	// Start ALL processors at once
	fmt.Println("Starting all batch processors...")
	batchClient.StartAll()

	// Add an event
	eventMessage := batch.Message{
		Data: &types.EventTrackRequest{
			Email:     "user1@example.com",
			EventName: "purchase",
			DataFields: map[string]interface{}{
				"productId": "prod123",
				"price":     29.99,
			},
		},
		MetaData: "event-1",
	}
	batchClient.EventTrack().Add(eventMessage)
	fmt.Println("Added event to batch")

	// Add a user update
	userMessage := batch.Message{
		Data: &types.BulkUpdateUser{
			Email:  "user2@example.com",
			UserId: "user2",
			DataFields: map[string]interface{}{
				"firstName": "Alice",
				"lastName":  "Smith",
			},
			PreferUserId:       true,
			MergeNestedObjects: &[]bool{true}[0],
		},
		MetaData: "user-update-1",
	}
	batchClient.UserUpdate().Add(userMessage)
	fmt.Println("Added user update to batch")

	// Add a subscription update
	subscriptionMessage := batch.Message{
		Data: &types.UserUpdateSubscriptionsRequest{
			Email:                    "user3@example.com",
			EmailListIds:             []int64{12345}, // Replace with actual list ID
			SubscribedMessageTypeIds: []int64{111},   // Replace with actual message type ID
		},
		MetaData: "subscription-update-1",
	}
	batchClient.SubscriptionUpdate().Add(subscriptionMessage)
	fmt.Println("Added subscription update to batch")

	// Listen for responses in a separate goroutine
	responseCount := 0
	expectedResponses := 3

	go func() {
		for response := range responseChan {
			responseCount++
			if response.Error != nil {
				fmt.Printf("Operation failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("Operation successful (MetaData: %v)\n", response.OriginalReq.MetaData)
			}

			// Close channel when we've received all expected responses
			if responseCount >= expectedResponses {
				close(responseChan)
				return
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("Waiting for batch processing...")
	time.Sleep(8 * time.Second)

	// Stop ALL processors at once
	fmt.Println("Stopping all batch processors...")
	batchClient.StopAll()

	// Wait a bit more to ensure all responses are processed
	time.Sleep(2 * time.Second)

	fmt.Println("All batch operations completed!")
}
