package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
)

func api_events_get_by_email(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	email := "user@example.com"
	events, err := client.Events().GetByEmail(email)
	if err != nil {
		log.Fatalf("Error getting events by email: %v", err)
	}

	fmt.Printf("Found %d events for user %s:\n", len(events), email)
	for _, event := range events {
		fmt.Printf("Event: %v\n", event)
	}
}
