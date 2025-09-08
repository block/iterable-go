package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
	"github.com/block/iterable-go/types"
)

func api_events_track(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	req := types.EventTrackRequest{
		Email:     "user@example.com",
		EventName: "purchase",
		DataFields: map[string]interface{}{
			"productId":   "prod123",
			"productName": "Widget",
			"price":       29.99,
			"quantity":    2,
			"category":    "electronics",
		},
		UserId: "user123",
	}

	res, err := client.Events().Track(req)
	if err != nil {
		log.Fatalf("Error tracking event: %v", err)
	}

	fmt.Printf("Event tracking response:\n")
	fmt.Printf("  Message: %s\n", res.Message)
	fmt.Printf("  Code: %s\n", res.Code)
	fmt.Printf("  Params: %+v\n", res.Params)
}
