package types

type List struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"createdAt"`
	ListType    string `json:"listType"`
}

type ListsResponse struct {
	Lists []List `json:"lists"`
}

type ListCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListCreateResponse struct {
	ListId int64 `json:"listId"`
}

type ListSubscriber struct {
	Email              string                 `json:"email,omitempty"`
	DataFields         map[string]interface{} `json:"dataFields,omitempty"`
	UserId             string                 `json:"userId,omitempty"`
	PreferUserId       bool                   `json:"preferUserId"`
	MergeNestedObjects bool                   `json:"mergeNestedObjects,omitempty"`
}

type ListSubscribeRequest struct {
	ListId      int64            `json:"listId"`
	Subscribers []ListSubscriber `json:"subscribers"`
	// Whether to skip operation when the request includes a userId or email
	// that doesn't yet exist in the Iterable project.
	// When true, Iterable ignores requests with unknown userIds and email addresses.
	// When false, Iterable creates new users.
	// Only respected in API calls for userID-based and hybrid projects.
	// Defaults to false.
	UpdateExistingUsersOnly bool `json:"updateExistingUsersOnly,omitempty"`
}

type ListSubscribeResponse struct {
	SuccessCount      int64         `json:"successCount"`
	FailCount         int64         `json:"failCount"`
	FailedUpdates     FailedUpdates `json:"failedUpdates"`
	FilteredOutFields []string      `json:"filteredOutFields"`
	CreatedFields     []string      `json:"createdFields"`
}

type ListUnSubscriber struct {
	Email  string `json:"email,omitempty"`
	UserId string `json:"userId,omitempty"`
}

type ListUnSubscribeRequest struct {
	ListId             int64              `json:"listId"`
	Subscribers        []ListUnSubscriber `json:"subscribers"`
	CampaignId         int64              `json:"campaignId"`
	ChannelUnsubscribe bool               `json:"channelUnsubscribe"`
}

type ListUnSubscribeResponse struct {
	SuccessCount      int64         `json:"successCount"`
	FailCount         int64         `json:"failCount"`
	FailedUpdates     FailedUpdates `json:"failedUpdates"`
	FilteredOutFields []string      `json:"filteredOutFields"`
	CreatedFields     []string      `json:"createdFields"`
}
