package model

import "time"

// https://www.w3.org/TR/websub/#x5-1-subscriber-sends-subscription-request
type SubscribeRequest struct {
	Callback     string
	Mode         string
	Topic        string
	Secret       string
	LeaseSeconds int
}

type Subscription struct {
	Callback string
	Topic    string
	Secret   string
	Expires  time.Time
}

type SubscriptionIntentRequest struct {
	Mode         string
	Topic        string
	Challenge    string
	LeaseSeconds int
}
