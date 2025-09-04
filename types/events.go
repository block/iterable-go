package types

type Event struct {
	EventName      string                 `json:"eventName"`
	Email          string                 `json:"email"`
	CreatedAt      string                 `json:"createdAt"`
	EventUpdatedAt string                 `json:"eventUpdatedAt"`
	ItblInternal   map[string]interface{} `json:"itblInternal"`
	EventType      string                 `json:"eventType"`
	DataFields     map[string]interface{} `json:"dataFields"`
}

type EventTrackRequest struct {
	Email           string                 `json:"email,omitempty"`
	UserId          string                 `json:"userId,omitempty"`
	EventName       string                 `json:"eventName"`
	Id              string                 `json:"id,omitempty"`
	CreatedAt       int64                  `json:"createdAt"`
	DataFields      map[string]interface{} `json:"dataFields"`
	CampaignId      int64                  `json:"campaignId,omitempty"`
	TemplateId      int64                  `json:"templateId,omitempty"`
	CreateNewFields *bool                  `json:"createNewFields,omitempty"`
}

type EventTrackBulkRequest struct {
	Events []EventTrackRequest `json:"events"`
}

type EventsResponse struct {
	Events []Event `json:"events"`
}

type BulkEventsResponse struct {
	SuccessCount         int64              `json:"successCount"`
	FailCount            int64              `json:"failCount"`
	DisallowedEventNames []string           `json:"disallowedEventNames"`
	FilteredOutFields    []string           `json:"filteredOutFields"`
	CreatedFields        []string           `json:"createdFields"`
	FailedUpdates        FailedEventUpdates `json:"failedUpdates"`
}

type FailedEventUpdates struct {
	InvalidEmails    []string `json:"invalidEmails"`
	InvalidUserIds   []string `json:"invalidUserIds"`
	NotFoundEmails   []string `json:"notFoundEmails"`
	NotFoundUserIds  []string `json:"notFoundUserIds"`
	ForgottenEmails  []string `json:"forgottenEmails"`
	ForgottenUserIds []string `json:"forgottenUserIds"`
}
