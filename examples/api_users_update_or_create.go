package main

import (
	"fmt"
	"log"

	iterable_go "iterable-go"
	"iterable-go/types"
)

func api_users_update_or_create(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	req := types.UserRequest{
		Email:  "user@example.com",
		UserId: "user123",
		DataFields: map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       30,
			"city":      "San Francisco",
		},
		PreferUserId:       true,
		MergeNestedObjects: &[]bool{true}[0], // Helper to get pointer to bool
	}

	res, err := client.Users().UpdateOrCreate(req)
	if err != nil {
		log.Fatalf("Error updating/creating user: %v", err)
	}

	fmt.Printf("User update/create response:\n")
	fmt.Printf("  Message: %s\n", res.Message)
	fmt.Printf("  Code: %s\n", res.Code)
	fmt.Printf("  Params: %+v\n", res.Params)
}
