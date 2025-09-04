package types

import "time"

type Campaign struct {
	Id                  int64    `json:"id"`
	CreatedAt           int64    `json:"createdAt"`
	UpdatedAt           int64    `json:"updatedAt"`
	StartAt             int64    `json:"startAt,omitempty"`
	EndedAt             int64    `json:"endedAt,omitempty"`
	Name                string   `json:"name"`
	TemplateId          int64    `json:"templateId,omitempty"`
	MessageMedium       string   `json:"messageMedium"`
	CreatedByUserId     string   `json:"createdByUserId"`
	UpdatedByUserId     string   `json:"updatedByUserId,omitempty"`
	CampaignState       string   `json:"campaignState"`
	ListIds             []int64  `json:"listIds,omitempty"`
	SuppressionListIds  []int64  `json:"suppressionListIds,omitempty"`
	SendSize            int64    `json:"sendSize,omitempty"`
	RecurringCampaignId int64    `json:"recurringCampaignId,omitempty"`
	WorkflowId          int64    `json:"workflowId,omitempty"`
	Labels              []string `json:"labels,omitempty"`
	Type                string   `json:"type"`
}

type CampaignsResponse struct {
	Campaigns []Campaign `json:"campaigns"`
}

type camSendMode struct{}

func (camSendMode) RecipientTimeZone() string {
	return "RecipientTimeZone"
}
func (camSendMode) ProjectTimeZone() string {
	return "ProjectTimeZone"
}
func (cam camSendMode) Default() string {
	return cam.ProjectTimeZone()
}

var CampaignSendMode = camSendMode{}

func CampaignSentAtFormat(d time.Time) string {
	return d.UTC().Format("2006-01-02 15:04:05")
}

type CreateCampaignRequest struct {
	Name               string         `json:"name"`
	ListIds            []int64        `json:"listIds"`
	TemplateId         int64          `json:"templateId"`
	SuppressionListIds []int64        `json:"suppressionListIds"`
	SendAt             string         `json:"sendAt"`
	SendMode           string         `json:"sendMode"`
	StartTimeZone      string         `json:"startTimeZone"`
	DefaultTimeZone    string         `json:"defaultTimeZone"`
	DataFields         map[string]any `json:"dataFields"`
}

type campaignId struct {
	CampaignId int64 `json:"campaignId"`
}

type CreateCampaignResponse = campaignId
type AbortCampaignRequest = campaignId
type CancelCampaignRequest = campaignId

type TriggerCampaignRequest struct {
	AllowRepeatMarketingSends *bool          `json:"allowRepeatMarketingSends,omitempty"`
	CampaignId                int64          `json:"campaignId"`
	DataFields                map[string]any `json:"dataFields,omitempty"`
	ListIds                   []int64        `json:"listIds,omitempty"`
	SuppressionListIds        []int64        `json:"suppressionListIds,omitempty"`
}

type ActivateTriggeredCampaignRequest = campaignId
