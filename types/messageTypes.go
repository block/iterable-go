package types

type MessageType struct {
	Id                 int          `json:"id"`
	CreatedAt          int          `json:"createdAt"`
	Name               string       `json:"name"`
	ChannelId          int          `json:"channelId"`
	SubscriptionPolicy string       `json:"subscriptionPolicy"`
	RateLimitPerMinute int          `json:"rateLimitPerMinute"`
	FrequencyCap       FrequencyCap `json:"frequencyCap"`
}

type FrequencyCap struct {
	Days     int `json:"days"`
	Messages int `json:"messages"`
}

type MessageTypeResponse struct {
	MessageTypes []MessageType `json:"messageTypes"`
}
