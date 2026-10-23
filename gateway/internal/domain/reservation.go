package domain

import "github.com/google/uuid"

type ReservationStatus string

const (
	StatusRented   ReservationStatus = "RENTED"
	StatusReturned ReservationStatus = "RETURNED"
	StatusExpired  ReservationStatus = "EXPIRED"
)

type Reservation struct {
	ReservationUID  uuid.UUID
	BookUID         uuid.UUID
	LibraryUID      uuid.UUID
	Status          ReservationStatus
	StartDate       Date
	TillDate        Date
	ConditionAtRent BookCondition
}

type NewReservation struct {
	BookUID         uuid.UUID
	LibraryUID      uuid.UUID
	TillDate        Date
	ConditionAtRent BookCondition
}

type ReservationDetails struct {
	Reservation Reservation
	Book        Book
	Library     Library
}

type TakenBook struct {
	ReservationDetails
	Rating Rating
}
