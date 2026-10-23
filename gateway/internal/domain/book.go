package domain

import "github.com/google/uuid"

type BookCondition string

const (
	ConditionExcellent BookCondition = "EXCELLENT"
	ConditionGood      BookCondition = "GOOD"
	ConditionBad       BookCondition = "BAD"
)

type Book struct {
	BookUID uuid.UUID
	Name    string
	Author  string
	Genre   string
}

type LibraryBook struct {
	Book           Book
	Condition      BookCondition
	AvailableCount int
}

type BookCopy struct {
	Condition      BookCondition
	AvailableCount int
}
