package domain

import "errors"

var (
	ErrLibraryNotFound     = errors.New("library not found")
	ErrBookNotFound        = errors.New("book not found")
	ErrNoAvailableCopies   = errors.New("no available copies")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrReservationClosed   = errors.New("reservation is already closed")
	ErrBookLimitReached    = errors.New("book limit reached")
)
