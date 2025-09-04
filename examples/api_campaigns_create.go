package main

import (
	"fmt"
	"log"
	"time"

	iterable_go "iterable-go"
	"iterable-go/types"
)

func api_campaigns_create(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	listId := int64(1000)
	list2Id := int64(1001)
	templateId := int64(1002)
	campaignId, err := client.Campaigns().Create(types.CreateCampaignRequest{
		Name:               "test-campaign",
		ListIds:            []int64{listId},
		SuppressionListIds: []int64{list2Id},
		TemplateId:         templateId,
		SendMode:           types.CampaignSendMode.ProjectTimeZone(),
		SendAt:             types.CampaignSentAtFormat(time.Now()),
		DataFields: map[string]any{
			"{{unsubscribeUrl}}":          "example.com/unsub",
			"{{hostedUnsubscribeUrl":      "example.com/hosted",
			"{{unsubscribeMessageTypeUrl": "example.com/unsubMessage",
		},
	})

	if err != nil {
		log.Fatalf("Error creating new Campaign: %v", err)
	}

	fmt.Printf("New Campaign created: %d", campaignId)
}
