package utils

import "github.com/google/uuid"

// NewID generates a new random unique identifier for domain records.
func NewID() string {
	return uuid.NewString()
}
