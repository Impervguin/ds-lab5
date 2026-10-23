package dto

import "github.com/Impervguin/ds-lab2/gateway/internal/domain"

type UserRatingResponse struct {
	Stars    int `json:"stars"`
	MaxBooks int `json:"maxBooks"`
}

func NewUserRatingResponse(rating domain.Rating) UserRatingResponse {
	return UserRatingResponse{Stars: rating.Stars, MaxBooks: rating.MaxBooks}
}
