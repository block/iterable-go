package main

import (
	"fmt"
	"log"

	iterable_go "iterable-go"
)

func api_channels_get_all(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	channels, err := client.Channels().Channels()
	if err != nil {
		log.Fatalf("Error getting channels: %v", err)
	}

	fmt.Printf("Found %d channels:\n", len(channels))
	for _, channel := range channels {
		fmt.Printf("Channel: %v\n", channel)
	}
}
