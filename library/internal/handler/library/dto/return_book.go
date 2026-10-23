package dto

import "github.com/Impervguin/ds-lab2/library/internal/domain"

type ReturnBookRequest struct {
	Condition string `json:"condition" validate:"required,oneof=EXCELLENT GOOD BAD"`
}

type ReturnBookResponse struct {
	Condition      string `json:"condition"`
	AvailableCount int    `json:"availableCount"`
}

func NewReturnBookResponse(returned domain.LibraryBook) ReturnBookResponse {
	return ReturnBookResponse{
		Condition:      string(returned.Condition),
		AvailableCount: returned.AvailableCount,
	}
}
