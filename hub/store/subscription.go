package store

import (
	"client/model"
	"sync"
	"time"
)

/*
A store to keep track of subscriptions.
The store is a map of topic (str) to a map of callback (str) to subscription.
*/
type SubscriptionStore struct {
	subscriptions map[string]map[string]model.Subscription
	sync.Mutex
}

func NewSubscriptionStore() *SubscriptionStore {
	return &SubscriptionStore{subscriptions: make(map[string]map[string]model.Subscription)}
}

func (ss *SubscriptionStore) AddSubscription(topic string, callback string, secret string, leaseSeconds int) {
	ss.Lock()
	defer ss.Unlock()

	// Check if the topic exists, if not create it
	if _, ok := ss.subscriptions[topic]; !ok {
		ss.subscriptions[topic] = make(map[string]model.Subscription)
	}

	expires := time.Now().Add(time.Duration(leaseSeconds) * time.Second)

	ss.subscriptions[topic][callback] = model.Subscription{Callback: callback, Topic: topic, Secret: secret, Expires: expires}
}

func (ss *SubscriptionStore) RemoveSubscription(topic string, callback string) {
	ss.Lock()
	defer ss.Unlock()

	if callbacks, ok := ss.subscriptions[topic]; ok {
		delete(callbacks, callback)
		if len(callbacks) == 0 {
			delete(ss.subscriptions, topic) // Remove the topic if no callbacks are left
		}
	}
}

// Get all subscriptions for a topic
func (ss *SubscriptionStore) GetSubscriptions(topic string) []model.Subscription {
	ss.Lock()
	defer ss.Unlock()

	subscriptions := make([]model.Subscription, 0, len(ss.subscriptions[topic]))
	for _, s := range ss.subscriptions[topic] {
		subscriptions = append(subscriptions, s)
	}
	return subscriptions
}

func (ss *SubscriptionStore) RemoveOutdatedSubscriptions() {
	ss.Lock()
	defer ss.Unlock()

	now := time.Now()
	for topic, callbacks := range ss.subscriptions {
		for callback, s := range callbacks {
			if s.Expires.Before(now) {
				delete(callbacks, callback)
			}
		}
		if len(callbacks) == 0 {
			delete(ss.subscriptions, topic)
		}
	}
}
