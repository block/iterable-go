package main

import (
	"fmt"
	"time"

	iterable_go "iterable-go"
	"iterable-go/batch"
	"iterable-go/types"
)

func batch_email_update(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	responseChan := make(chan batch.Response, 100)

	batchClient := iterable_go.NewBatch(client,
		iterable_go.WithBatchFlushQueueSize(1),              // Process immediately (individual processing)
		iterable_go.WithBatchFlushInterval(1*time.Second),   // Process every second
		iterable_go.WithBatchResponseListener(responseChan), // Send responses to this channel
	)

	batchClient.EmailUpdate().Start()

	emailUpdates := []types.UserUpdateEmailRequest{
		{
			Email:    "olduser1@example.com",
			UserId:   "user1",
			NewEmail: "newuser1@example.com",
		},
		{
			Email:    "olduser2@example.com",
			UserId:   "user2",
			NewEmail: "newuser2@example.com",
		},
		{
			Email:    "olduser3@example.com",
			UserId:   "user3",
			NewEmail: "newuser3@example.com",
		},
	}

	for i, update := range emailUpdates {
		message := batch.Message{
			Data:     &update,
			MetaData: fmt.Sprintf("email-update-%d", i+1), // Optional tracking ID
		}
		batchClient.EmailUpdate().Add(message)
		fmt.Printf("Added email update %d: %s -> %s (User: %s)\n",
			i+1, update.Email, update.NewEmail, update.UserId)
	}

	// Listen for responses in a separate goroutine
	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("Email update failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("Email updated successfully (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete
	fmt.Println("Waiting for email updates to process...")
	fmt.Println("Note: Email updates are processed individually, not in batches")
	time.Sleep(10 * time.Second)

	// Stop the processor
	batchClient.EmailUpdate().Stop()
	close(responseChan)

	fmt.Println("Email update processing completed!")
}
