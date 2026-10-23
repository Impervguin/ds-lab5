package domain

import (
	"context"

	"github.com/google/uuid"
)

type BookCondition string

const (
	ConditionExcellent BookCondition = "EXCELLENT"
	ConditionGood      BookCondition = "GOOD"
	ConditionBad       BookCondition = "BAD"
)

var Conditions = []BookCondition{ConditionExcellent, ConditionGood, ConditionBad}

func (c BookCondition) Valid() bool {
	for _, known := range Conditions {
		if c == known {
			return true
		}
	}
	return false
}

func (c BookCondition) Rank() int {
	for rank, known := range Conditions {
		if c == known {
			return rank
		}
	}
	return len(Conditions)
}

type Book struct {
	ID      int64
	BookUID uuid.UUID
	Name    string
	Author  string
	Genre   string
}

type LibraryBook struct {
	Book           Book
	LibraryID      int64
	Condition      BookCondition
	AvailableCount int
}

type BookRepository interface {
	Get(ctx context.Context, bookUID uuid.UUID) (*Book, error)
}

type LibraryBooksUpdateFunc func(ctx context.Context, held []LibraryBook) ([]LibraryBook, error)

type LibraryBookRepository interface {
	ListByLibrary(ctx context.Context, libraryUID uuid.UUID, limit, offset int, includeEmpty bool) ([]LibraryBook, int, error)
	Update(ctx context.Context, libraryUID, bookUID uuid.UUID, updFunc LibraryBooksUpdateFunc) ([]LibraryBook, error)
}
