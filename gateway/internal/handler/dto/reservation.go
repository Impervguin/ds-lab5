package dto

import "github.com/Impervguin/ds-lab2/gateway/internal/domain"

type TakeBookRequest struct {
	BookUID    string `json:"bookUid" validate:"required,uuid"`
	LibraryUID string `json:"libraryUid" validate:"required,uuid"`
	TillDate   string `json:"tillDate" validate:"required,datetime=2006-01-02"`
}

type ReturnBookRequest struct {
	Condition string `json:"condition" validate:"required,oneof=EXCELLENT GOOD BAD"`
	Date      string `json:"date" validate:"required,datetime=2006-01-02"`
}

type BookReservationResponse struct {
	ReservationUID string          `json:"reservationUid"`
	Status         string          `json:"status"`
	StartDate      string          `json:"startDate"`
	TillDate       string          `json:"tillDate"`
	Book           BookResponse    `json:"book"`
	Library        LibraryResponse `json:"library"`
}

type TakeBookResponse struct {
	BookReservationResponse
	Rating UserRatingResponse `json:"rating"`
}

func NewBookReservationResponse(details domain.ReservationDetails) BookReservationResponse {
	return BookReservationResponse{
		ReservationUID: details.Reservation.ReservationUID.String(),
		Status:         string(details.Reservation.Status),
		StartDate:      details.Reservation.StartDate.String(),
		TillDate:       details.Reservation.TillDate.String(),
		Book:           NewBookResponse(details.Book),
		Library:        NewLibraryResponse(details.Library),
	}
}

func NewBookReservationResponses(details []domain.ReservationDetails) []BookReservationResponse {
	reservations := make([]BookReservationResponse, 0, len(details))
	for _, reservation := range details {
		reservations = append(reservations, NewBookReservationResponse(reservation))
	}
	return reservations
}

func NewTakeBookResponse(taken domain.TakenBook) TakeBookResponse {
	return TakeBookResponse{
		BookReservationResponse: NewBookReservationResponse(taken.ReservationDetails),
		Rating:                  NewUserRatingResponse(taken.Rating),
	}
}
