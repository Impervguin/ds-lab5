package dto

import "github.com/Impervguin/ds-lab2/library/internal/domain"

type TakeBookResponse struct {
	Condition      string `json:"condition"`
	AvailableCount int    `json:"availableCount"`
}

func NewTakeBookResponse(taken domain.LibraryBook) TakeBookResponse {
	return TakeBookResponse{
		Condition:      string(taken.Condition),
		AvailableCount: taken.AvailableCount,
	}
}
