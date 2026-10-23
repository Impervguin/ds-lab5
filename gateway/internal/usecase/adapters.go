package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
)

type LibraryService interface {
	ListLibraries(ctx context.Context, city string, page, size int) (Page[domain.Library], error)
	GetLibrary(ctx context.Context, libraryUID uuid.UUID) (*domain.Library, error)
	ListBooks(ctx context.Context, libraryUID uuid.UUID, page, size int, showAll bool) (Page[domain.LibraryBook], error)
	GetBook(ctx context.Context, bookUID uuid.UUID) (*domain.Book, error)
	TakeBook(ctx context.Context, libraryUID, bookUID uuid.UUID, inQueue bool) (*domain.BookCopy, error)
	ReturnBook(ctx context.Context, libraryUID, bookUID uuid.UUID, condition domain.BookCondition, inQueue bool) (*domain.BookCopy, error)
}

type ReservationService interface {
	ListReservations(ctx context.Context, status domain.ReservationStatus) ([]domain.Reservation, error)
	CountReservations(ctx context.Context, status domain.ReservationStatus) (int, error)
	CreateReservation(ctx context.Context, request domain.NewReservation, inQueue bool) (*domain.Reservation, error)
	ReturnReservation(ctx context.Context, reservationUID uuid.UUID, date domain.Date, inQueue bool) (*domain.Reservation, error)
}

type RatingService interface {
	GetRating(ctx context.Context) (*domain.Rating, error)
	CloseReservation(ctx context.Context, closed domain.ClosedReservation, inQueue bool) (*domain.RatingChange, error)
}

var ErrServiceUnavailable = errors.New("service unavailable")

var (
	ErrLibraryUnavailable     = fmt.Errorf("library %w", ErrServiceUnavailable)
	ErrReservationUnavailable = fmt.Errorf("reservation %w", ErrServiceUnavailable)
	ErrRatingUnavailable      = fmt.Errorf("rating %w", ErrServiceUnavailable)
)
var ErrQueued = errors.New("queued")

// unavailable tags an ErrServiceUnavailable the use case could not handle
// with the upstream it came from; any other error is returned untouched.
func unavailable(err, service error) error {
	if errors.Is(err, ErrServiceUnavailable) {
		return fmt.Errorf("%w: %w", service, err)
	}
	return err
}
