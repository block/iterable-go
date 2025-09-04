package main

import (
	"fmt"
	"time"

	iterable_go "iterable-go"
	"iterable-go/batch"
	"iterable-go/types"
)

func batch_user_update(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	// Create a batch client with custom configuration
	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchFlushQueueSize(5),              // Process when 5 users accumulate
		iterable_go.WithBatchFlushInterval(3*time.Second),   // Or process every 3 seconds
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	// Start the user update processor
	batchClient.UserUpdate().Start()

	// Add multiple user updates for batch processing
	users := []types.BulkUpdateUser{
		{
			Email:  "user1@example.com",
			UserId: "user1",
			DataFields: map[string]interface{}{
				"firstName": "Alice",
				"lastName":  "Smith",
				"age":       25,
				"city":      "New York",
				"plan":      "premium",
			},
			PreferUserId:       true,
			MergeNestedObjects: &[]bool{true}[0],
		},
		{
			Email:  "user2@example.com",
			UserId: "user2",
			DataFields: map[string]interface{}{
				"firstName": "Bob",
				"lastName":  "Johnson",
				"age":       35,
				"city":      "Los Angeles",
				"plan":      "basic",
			},
			PreferUserId:       true,
			MergeNestedObjects: &[]bool{true}[0],
		},
		{
			Email:  "user3@example.com",
			UserId: "user3",
			DataFields: map[string]interface{}{
				"firstName": "Carol",
				"lastName":  "Davis",
				"age":       28,
				"city":      "Chicago",
				"plan":      "premium",
			},
			PreferUserId:       true,
			MergeNestedObjects: &[]bool{true}[0],
		},
	}

	// Add users to the batch processor
	for i, user := range users {
		message := batch.Message{
			Data:     &user,
			MetaData: fmt.Sprintf("user-update-%d", i+1), // Optional tracking ID
		}
		batchClient.UserUpdate().Add(message)
		fmt.Printf("Added user %d to batch: %s (%s)\n", i+1, user.Email, user.UserId)
	}

	// Listen for responses in a separate goroutine
	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("User update failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("User updated successfully (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("Waiting for batch processing...")
	time.Sleep(8 * time.Second)

	// Stop the processor
	batchClient.UserUpdate().Stop()
	close(responseChan)

	fmt.Println("Batch user update completed!")
}
