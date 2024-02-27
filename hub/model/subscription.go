package model

// https://www.w3.org/TR/websub/#x5-1-subscriber-sends-subscription-request
// Lease seconds is also part of the websub subscription request, but its not used in this case
type SubscribeRequest struct {
	Callback string
	Mode     string
	Topic    string
	Secret   string
}

type Subscription struct {
	Callback string
	Topic    string
	Secret   string
}
