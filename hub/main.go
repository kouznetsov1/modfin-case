package main

import (
	"bytes"
	"client/model"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var subscriptions []model.Subscription

func main() {
	http.HandleFunc("/", handleSubscribe)

	// Start a ticker to notify subscribers every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				notifySubscribers()
			}
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
func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sr := model.SubscribeRequest{
		Callback: r.FormValue("hub.callback"),
		Mode:     r.FormValue("hub.mode"),
		Topic:    r.FormValue("hub.topic"),
		Secret:   r.FormValue("hub.secret"),
	}

	// Generate string and check that subscriber echoes it back
	// https://www.w3.org/TR/websub/#x5-3-hub-verifies-intent-of-the-subscriber
	challenge := generateRandomString(16)
	verificationUrl := fmt.Sprintf("%s?hub.mode=%s&hub.topic=%s&hub.challenge=%s", sr.Callback, sr.Mode, sr.Topic, challenge)

	res, err := http.Get(verificationUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer res.Body.Close()

	// Read body to check if subscriber echoed challenge
	b, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If the response status code is not 200 or subscriber did not echo challenge, return an error
	if res.StatusCode != http.StatusOK || string(b) != challenge {
		http.Error(w, "Failed to verify intent of subscriber", http.StatusForbidden)
		return
	}

	// Add to global list of subscriptions if mode is "subscribe"
	if sr.Mode == "subscribe" {
		subscriptions = append(subscriptions, model.Subscription{Callback: sr.Callback, Topic: sr.Topic, Secret: sr.Secret})
	}

	// Remove from global list of subscriptions if mode is "unsubscribe"
	if sr.Mode == "unsubscribe" {
		for i, sub := range subscriptions {
			if sub.Callback == sr.Callback {
				subscriptions = append(subscriptions[:i], subscriptions[i+1:]...)
				break
			}
		}
	}
}

// Sends a notification to all subscribers of a topic
func notifySubscribers() {
	b := []byte(fmt.Sprintf(`{"data": "some data", "date": "%s"}`, time.Now().Format("2006-01-02 12:12:12")))

	for _, sub := range subscriptions {
		// Sign the data
		hash := sign(sub, string(b))

		req, err := http.NewRequest(http.MethodPost, sub.Callback, bytes.NewReader(b))
		if err != nil {
			fmt.Printf("Error creating request: %s\n", err)
			return
		}
		req.Header.Set("X-Hub-Signature", fmt.Sprintf("sha256=%s", hash))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("error sending notification: %s\n", err)
			continue
		}
		resp.Body.Close()
	}

}

// Generates a new HMAC signature for sending notification
// Uses SHA256, secret as key and the request body as data
func sign(s model.Subscription, b string) string {
	hash := hmac.New(sha256.New, []byte(s.Secret))
	hash.Write([]byte(b))
	return hex.EncodeToString(hash.Sum(nil))
}
