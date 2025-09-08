package main

import (
	"fmt"
	"time"

	iterable_go "github.com/block/iterable-go"
	"github.com/block/iterable-go/batch"
	"github.com/block/iterable-go/types"
)

func batch_custom_config(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	batchClient.EventTrack().Start()

	events := []types.EventTrackRequest{
		{
			Email:     "user1@example.com",
			EventName: "login",
			DataFields: map[string]interface{}{
				"source": "web",
				"device": "desktop",
			},
		},
		{
			Email:     "user2@example.com",
			EventName: "purchase",
			DataFields: map[string]interface{}{
				"productId": "prod123",
				"amount":    99.99,
			},
		},
		{
			Email:     "user3@example.com",
			EventName: "page_view",
			DataFields: map[string]interface{}{
				"page":       "/dashboard",
				"time_spent": 45,
			},
		},
	}

	// Add events to trigger batch processing
	for i, event := range events {
		message := batch.Message{
			Data:     &event,
			MetaData: fmt.Sprintf("custom-config-event-%d", i+1),
		}
		batchClient.EventTrack().Add(message)
		fmt.Printf("Added event %d: %s for %s\n", i+1, event.EventName, event.Email)
	}

	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("❌ Event failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("✅ Event successful (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("\nWaiting for batch processing with custom configuration...")
	time.Sleep(10 * time.Second)

	batchClient.EventTrack().Stop()
	close(responseChan)

	fmt.Println("\nBatch processing with custom configuration completed!")
}
