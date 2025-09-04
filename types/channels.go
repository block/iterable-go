package types

type Channel struct {
	Id            int64  `json:"id"`
	Name          string `json:"name"`
	ChannelType   string `json:"channelType"`
	MessageMedium string `json:"messageMedium"`
}

type ChannelsResponse struct {
	Channels []Channel `json:"channels"`
}
