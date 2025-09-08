package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
	"github.com/block/iterable-go/types"
)

func api_lists_subscribe(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	req := types.ListSubscribeRequest{
		ListId: 12345,
		Subscribers: []types.ListSubscriber{
			{
				Email:  "user@example.com",
				UserId: "user123",
				DataFields: map[string]interface{}{
					"firstName": "John",
					"lastName":  "Doe",
					"source":    "api_subscription",
				},
			},
		},
	}

	res, err := client.Lists().Subscribe(req)
	if err != nil {
		log.Fatalf("Error subscribing to list: %v", err)
	}

	fmt.Printf("List subscription response:\n")
	fmt.Printf("  Success Count: %d\n", res.SuccessCount)
	fmt.Printf("  Fail Count: %d\n", res.FailCount)
	fmt.Printf("  FailedUpdates: %v\n", res.FailedUpdates)
	fmt.Printf("  Created Fields: %v\n", res.CreatedFields)
}
