package domain

import (
	"context"

	"github.com/google/uuid"
)

type Library struct {
	ID         int64
	LibraryUID uuid.UUID
	Name       string
	City       string
	Address    string
}

type LibraryRepository interface {
	ListByCity(ctx context.Context, city string, limit, offset int) ([]Library, int, error)
	Get(ctx context.Context, libraryUID uuid.UUID) (*Library, error)
}
