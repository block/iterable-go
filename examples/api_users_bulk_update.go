package main

import (
	"fmt"
	"log"

	iterable_go "iterable-go"
	"iterable-go/types"
)

func api_users_bulk_update(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	req := types.BulkUpdateRequest{
		Users: []types.BulkUpdateUser{
			{
				Email:  "user1@example.com",
				UserId: "user1",
				DataFields: map[string]interface{}{
					"firstName": "Alice",
					"lastName":  "Smith",
					"age":       25,
					"city":      "New York",
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
				},
				PreferUserId:       true,
				MergeNestedObjects: &[]bool{true}[0],
			},
		},
		CreateNewFields: &[]bool{true}[0],
	}

	res, err := client.Users().BulkUpdate(req)
	if err != nil {
		log.Fatalf("Error bulk updating users: %v", err)
	}

	fmt.Printf("Bulk update response:\n")
	fmt.Printf("  Success Count: %d\n", res.SuccessCount)
	fmt.Printf("  Fail Count: %d\n", res.FailCount)
	fmt.Printf("  Fail Count: %v\n", res.FailedUpdates)
	fmt.Printf("  Created Fields: %v\n", res.CreatedFields)
	fmt.Printf("  Filtered Out Fields: %v\n", res.FilteredOutFields)
}
