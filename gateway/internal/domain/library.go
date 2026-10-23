package domain

import "github.com/google/uuid"

type Library struct {
	LibraryUID uuid.UUID
	Name       string
	City       string
	Address    string
}
