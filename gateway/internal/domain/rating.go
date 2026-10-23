package domain

import "github.com/google/uuid"

type Rating struct {
	Stars    int
	MaxBooks int
}

type RatingChange struct {
	Delta int
	Stars int
}

type ClosedReservation struct {
	ReservationUID    uuid.UUID
	Status            ReservationStatus
	ConditionAtRent   BookCondition
	ConditionOnReturn BookCondition
}
