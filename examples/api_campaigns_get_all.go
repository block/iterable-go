package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
)

func api_campaigns_get_all(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	campaigns, err := client.Campaigns().All()
	if err != nil {
		log.Fatalf("Error getting campaigns: %v", err)
	}

	fmt.Printf("Found %d campaigns:\n", len(campaigns))
	for _, campaign := range campaigns {
		fmt.Printf("Campaign: %v\n", campaign)
	}
}
