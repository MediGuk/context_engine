package utils

import "github.com/google/uuid"

// GenerateID crea un identificador único universal (UUID)
func GenerateID() string {
	return uuid.New().String()
}
