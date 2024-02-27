package main

import (
	"crypto/rand"
	"encoding/base64"
)

// Generates a random string of length `length`
func generateRandomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}
