package types

type User struct {
	Email      string                 `json:"email"`
	UserId     string                 `json:"userId"`
	DataFields map[string]interface{} `json:"dataFields"`
}

type UserSentMessage struct {
	MessageId  string `json:"messageId"`
	CampaignId int64  `json:"campaignId"`
	TemplateId int64  `json:"templateId"`
	CreatedAt  string `json:"createdAt"`
}

type UserRequest struct {
	Email      string                 `json:"email,omitempty"`
	UserId     string                 `json:"userId"`
	DataFields map[string]interface{} `json:"dataFields,omitempty"`

	// Whether a new user should be created if the request includes
	// a userId that doesn't yet exist in the Iterable project.
	// Only respected in API calls for email-based projects.
	// Defaults to false.
	PreferUserId bool `json:"preferUserId"`

	// Merge top-level objects instead of overwriting them.
	// For example, if a user profile has data {"mySettings":{"mobile":true}}
	// and the request has data {"mySettings":{"email":true}},
	// merging results in {"mySettings":{"mobile":true,"email":true}}.
	// Defaults to false.
	MergeNestedObjects *bool `json:"mergeNestedObjects,omitempty"`

	// Whether new fields should be ingested and added to the schema.
	// Defaults to project's setting to allow or drop unrecognized fields.
	CreateNewFields *bool `json:"createNewFields,omitempty"`
}

type BulkUpdateUser struct {
	Email              string                 `json:"email,omitempty"`
	UserId             string                 `json:"userId,omitempty"`
	DataFields         map[string]interface{} `json:"dataFields,omitempty"`
	PreferUserId       bool                   `json:"preferUserId"`
	MergeNestedObjects *bool                  `json:"mergeNestedObjects,omitempty"`
}

type BulkUpdateRequest struct {
	Users           []BulkUpdateUser `json:"users"`
	CreateNewFields *bool            `json:"createNewFields,omitempty"`
}

type BulkUpdateResponse struct {
	SuccessCount      int64         `json:"successCount"`
	FailCount         int64         `json:"failCount"`
	FailedUpdates     FailedUpdates `json:"failedUpdates"`
	FilteredOutFields []string      `json:"filteredOutFields"`
	CreatedFields     []string      `json:"createdFields"`
}

type UserUpdateEmailRequest struct {
	Email    string `json:"currentEmail"`
	UserId   string `json:"currentUserId"`
	NewEmail string `json:"newEmail"`
}

type UserUpdateSubscriptionsRequest struct {
	Email                      string  `json:"email,omitempty"`
	UserId                     string  `json:"userId,omitempty"`
	EmailListIds               []int64 `json:"emailListIds,omitempty"`
	UnsubscribedChannelIds     []int64 `json:"unsubscribedChannelIds,omitempty"`
	UnsubscribedMessageTypeIds []int64 `json:"unsubscribedMessageTypeIds,omitempty"`
	SubscribedMessageTypeIds   []int64 `json:"subscribedMessageTypeIds,omitempty"`
	CampaignId                 int64   `json:"campaignId,omitempty"`
	TemplateId                 int64   `json:"templateId,omitempty"`
}

type BulkUserUpdateSubscriptionsRequest struct {
	UpdateSubscriptionsRequests []UserUpdateSubscriptionsRequest `json:"updateSubscriptionsRequests"`
}

type BulkUserUpdateSubscriptionsResponse struct {
	SuccessCount        int64    `json:"successCount"`
	FailCount           int64    `json:"failCount"`
	InvalidEmails       []string `json:"invalidEmails"`
	InvalidUserIds      []string `json:"invalidUserIds"`
	ValidEmailFailures  []string `json:"validEmailFailures"`
	ValidUserIdFailures []string `json:"validUserIdFailures"`
}

type UserResponse struct {
	User User `json:"user"`
}

type UserFieldsResponse struct {
	Fields map[string]string `json:"fields"`
}

type UserForgottenEmailsResponse struct {
	HashedEmails []string `json:"hashedEmails"`
}

type UserSentMessagesRequest struct {
	Email                string
	UserId               string
	Limit                int
	CampaignIds          []int64
	StartDate            string
	EndDate              string
	ExcludeBlastCampaign bool
	MessageMedium        string
}

type GdprRequest struct {
	Email  string `json:"email,omitempty"`
	UserId string `json:"userId,omitempty"`
}

type UserSentMessagesResponse struct {
	Messages []UserSentMessage `json:"messages"`
}
