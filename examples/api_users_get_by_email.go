package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
)

func api_users_get_by_email(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	email := "user@example.com"
	user, found, err := client.Users().GetByEmail(email)
	if err != nil {
		log.Fatalf("Error getting user by email: %v", err)
	}

	if !found {
		fmt.Printf("User with email %s not found\n", email)
		return
	}

	fmt.Printf("User found:\n")
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  User ID: %s\n", user.UserId)
	fmt.Printf("  Data Fields: %+v\n", user.DataFields)
}
