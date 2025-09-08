package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
)

func api_lists_get_all(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	lists, err := client.Lists().All()
	if err != nil {
		log.Fatalf("Error getting lists: %v", err)
	}

	fmt.Printf("Found %d lists:\n", len(lists))
	for _, list := range lists {
		fmt.Printf("List: %v\n", list)
	}
}
