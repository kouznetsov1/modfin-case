package main

import (
	"client/model"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

// Generates a random string of length `length`
func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		// Generate a random index for selecting a character from the charset
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

// Generates a new HMAC signature for sending notification
// Uses SHA256, secret as key and the request body as data
func sign(s model.Subscription, b string) string {
	hash := hmac.New(sha256.New, []byte(s.Secret))
	hash.Write([]byte(b))
	return hex.EncodeToString(hash.Sum(nil))
}
