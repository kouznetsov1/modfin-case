package main

import (
	"bytes"
	"client/model"
	"client/store"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

var LEASE_SECONDS = 60 * 60 * 24 * 10

// This and the ticker kind of emulates a publisher
var topics = store.NewTopicStore()

func main() {
	subscriptions := store.NewSubscriptionStore()
	topic := "oil-price"
	topics.AddTopic(topic)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			subscribe(w, r, subscriptions)
		} else {
			http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
		}
	})

	// Start a ticker to notify subscribers every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			notifySubscribers(&topic, subscriptions)
		}
	}()

	// Remove outdated subscriptions every 10 minutes
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			subscriptions.RemoveOutdatedSubscriptions()
		}
	}()

	err := http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

// Handles a POST request from a subscriber to subscribe to a topic
func subscribe(w http.ResponseWriter, r *http.Request, ss *store.SubscriptionStore) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Confirm that the request has been made
	fmt.Fprintf(w, `{"message": "Subscription request received and will be processed."}`)

	sr, err := validateRequest(r)
	if err != nil {
		client := &http.Client{
			Timeout: time.Second * 10, // Set timeout to 10 seconds
		}

		url := fmt.Sprintf("%s?hub.mode=denied&topic=%s&reason=%s", sr.Callback, sr.Topic, err.Error())

		res, err := client.Get(url)
		if err != nil {
			fmt.Printf("Error making GET request: %v\n", err)
			return
		}
		defer res.Body.Close()
	} else {
		// Check intent, if not 2xx status, return error
		if err := verifyIntent(sr); err != nil {
			fmt.Printf("Verification of intent failed: %s\n", err)
			return
		}

		if sr.Mode == "subscribe" {
			ss.AddSubscription(sr.Topic, sr.Callback, sr.Secret, sr.LeaseSeconds)
		} else if sr.Mode == "unsubscribe" {
			ss.RemoveSubscription(sr.Topic, sr.Callback)
		} else {
			http.Error(w, "Invalid mode", http.StatusBadRequest)
			return
		}
	}

}

func validateRequest(r *http.Request) (model.SubscribeRequest, error) {
	sr := model.SubscribeRequest{
		Callback: r.FormValue("hub.callback"),
		Mode:     r.FormValue("hub.mode"),
		Topic:    r.FormValue("hub.topic"),
		Secret:   r.FormValue("hub.secret"),
	}

	leaseSecondsStr := r.FormValue("hub.lease_seconds")
	if leaseSecondsStr == "" {
		sr.LeaseSeconds = LEASE_SECONDS // Default lease seconds if not provided.
	} else {
		var err error
		sr.LeaseSeconds, err = strconv.Atoi(leaseSecondsStr)
		if err != nil {
			return model.SubscribeRequest{
				Callback: sr.Callback,
				Topic:    sr.Topic,
			}, fmt.Errorf("invalid lease_seconds: %v", err)
		}
	}

	if !topics.HasTopic(sr.Topic) {
		return model.SubscribeRequest{
			Callback: sr.Callback,
			Topic:    sr.Topic,
		}, fmt.Errorf("topic does not exist: %s", sr.Topic)
	}

	return sr, nil
}

// Generate string and check that subscriber echoes it back
// https://www.w3.org/TR/websub/#x5-3-hub-verifies-intent-of-the-subscriber
func verifyIntent(sr model.SubscribeRequest) error {
	challenge, err := generateRandomString(16)
	if err != nil {
		return &model.VerificationError{Message: err.Error(), Code: 0}
	}

	verificationUrl := fmt.Sprintf("%s?hub.mode=%s&hub.topic=%s&hub.challenge=%s", sr.Callback, sr.Mode, sr.Topic, challenge)

	// Set a timeout for outbound requests if the subscriber does not respond
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	res, err := client.Get(verificationUrl)
	if err != nil {
		return &model.VerificationError{Message: err.Error(), Code: 0}
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return &model.VerificationError{Message: ("Failed to read response body"), Code: 0}
	}

	// If status is 2xx and subscriber echoed challenge
	if res.StatusCode >= 200 && res.StatusCode < 300 && string(b) == challenge {
		return nil
	}
	return &model.VerificationError{
		Message: "Subscriber did not echo the challenge correctly or returned a non-2xx status",
		Code:    res.StatusCode,
	}
}

// Sends a notification to all subscribers of a topic
func notifySubscribers(topic *string, subscriptions *store.SubscriptionStore) {
	// Data to send to subscribers
	b := []byte(fmt.Sprintf(`{"data": "This is the topic that you're subscribed to: %s"}`, *topic))

	// Set a timeout for outbound requests if the subscriber does not respond
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	fmt.Print("Sending notification to subscribers\n")

	// For all subscriptions of the given topic
	s := subscriptions.GetSubscriptions(*topic)
	for _, sub := range s {
		// Sign the request and send it to the subscriber
		hash := sign(sub, string(b))

		req, err := http.NewRequest(http.MethodPost, sub.Callback, bytes.NewReader(b))
		if err != nil {
			fmt.Printf("Error creating request: %s\n", err)
			return
		}
		req.Header.Set("X-Hub-Signature", fmt.Sprintf("sha256=%s", hash))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("error sending notification: %s\n", err)
			continue
		}
		resp.Body.Close()
	}
}
