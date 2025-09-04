package main

import (
	"fmt"
	"time"

	iterable_go "iterable-go"
	"iterable-go/batch"
	"iterable-go/types"
)

func batch_event_track(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	// Create a batch client with custom configuration
	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchFlushQueueSize(10),             // Process when 10 events accumulate
		iterable_go.WithBatchFlushInterval(5*time.Second),   // Or process every 5 seconds
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	// Start the event tracking processor
	batchClient.EventTrack().Start()

	// Add multiple events for batch processing
	events := []types.EventTrackRequest{
		{
			Email:     "user1@example.com",
			EventName: "purchase",
			DataFields: map[string]interface{}{
				"productId":   "prod123",
				"productName": "Widget A",
				"price":       29.99,
				"quantity":    1,
			},
		},
		{
			Email:     "user2@example.com",
			EventName: "purchase",
			DataFields: map[string]interface{}{
				"productId":   "prod456",
				"productName": "Widget B",
				"price":       49.99,
				"quantity":    2,
			},
		},
		{
			Email:     "user3@example.com",
			EventName: "page_view",
			DataFields: map[string]interface{}{
				"page":     "/products",
				"category": "electronics",
			},
		},
	}

	// Add events to the batch processor
	for i, event := range events {
		message := batch.Message{
			Data:     &event,
			MetaData: fmt.Sprintf("event-%d", i+1), // Optional tracking ID
		}
		batchClient.EventTrack().Add(message)
		fmt.Printf("Added event %d to batch: %s for %s\n", i+1, event.EventName, event.Email)
	}

	// Listen for responses in a separate goroutine
	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("Event processing failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("Event processed successfully (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("Waiting for batch processing...")
	time.Sleep(10 * time.Second)

	// Stop the processor
	batchClient.EventTrack().Stop()
	close(responseChan)

	fmt.Println("Batch event tracking completed!")
}
