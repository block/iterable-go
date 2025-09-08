package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
)

func api_message_types_get(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	messageTypes, err := client.MessageTypes().Get()
	if err != nil {
		log.Fatalf("Error getting message types: %v", err)
	}

	fmt.Printf("Found %d message types:\n", len(messageTypes.MessageTypes))
	for _, messageType := range messageTypes.MessageTypes {
		fmt.Printf("Message Type: %v\n", messageType)
	}
}
