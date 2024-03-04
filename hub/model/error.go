package model

import "fmt"

type VerificationError struct {
    Message string
    Code    int
}

func (e *VerificationError) Error() string {
    return fmt.Sprintf("Verification failed: %s (Code: %d)", e.Message, e.Code)
}