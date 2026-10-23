package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type LibraryService struct {
	mock.Mock
}

var _ usecase.LibraryService = (*LibraryService)(nil)

func (m *LibraryService) ListLibraries(
	ctx context.Context,
	city string,
	page, size int,
) (usecase.Page[domain.Library], error) {
	args := m.Called(ctx, city, page, size)
	libraries, _ := args.Get(0).(usecase.Page[domain.Library])
	return libraries, args.Error(1)
}

func (m *LibraryService) GetLibrary(ctx context.Context, libraryUID uuid.UUID) (*domain.Library, error) {
	args := m.Called(ctx, libraryUID)
	library, _ := args.Get(0).(*domain.Library)
	return library, args.Error(1)
}

func (m *LibraryService) ListBooks(
	ctx context.Context,
	libraryUID uuid.UUID,
	page, size int,
	showAll bool,
) (usecase.Page[domain.LibraryBook], error) {
	args := m.Called(ctx, libraryUID, page, size, showAll)
	books, _ := args.Get(0).(usecase.Page[domain.LibraryBook])
	return books, args.Error(1)
}

func (m *LibraryService) GetBook(ctx context.Context, bookUID uuid.UUID) (*domain.Book, error) {
	args := m.Called(ctx, bookUID)
	book, _ := args.Get(0).(*domain.Book)
	return book, args.Error(1)
}

func (m *LibraryService) TakeBook(
	ctx context.Context,
	libraryUID, bookUID uuid.UUID,
	inQueue bool,
) (*domain.BookCopy, error) {
	args := m.Called(ctx, libraryUID, bookUID, inQueue)
	taken, _ := args.Get(0).(*domain.BookCopy)
	return taken, args.Error(1)
}

func (m *LibraryService) ReturnBook(
	ctx context.Context,
	libraryUID, bookUID uuid.UUID,
	condition domain.BookCondition,
	inQueue bool,
) (*domain.BookCopy, error) {
	args := m.Called(ctx, libraryUID, bookUID, condition, inQueue)
	shelved, _ := args.Get(0).(*domain.BookCopy)
	return shelved, args.Error(1)
}

type ReservationService struct {
	mock.Mock
}

var _ usecase.ReservationService = (*ReservationService)(nil)

func (m *ReservationService) ListReservations(
	ctx context.Context,
	status domain.ReservationStatus,
) ([]domain.Reservation, error) {
	args := m.Called(ctx, status)
	reservations, _ := args.Get(0).([]domain.Reservation)
	return reservations, args.Error(1)
}

func (m *ReservationService) CountReservations(
	ctx context.Context,
	status domain.ReservationStatus,
) (int, error) {
	args := m.Called(ctx, status)
	return args.Int(0), args.Error(1)
}

func (m *ReservationService) CreateReservation(
	ctx context.Context,
	request domain.NewReservation,
	inQueue bool,
) (*domain.Reservation, error) {
	args := m.Called(ctx, request, inQueue)
	reservation, _ := args.Get(0).(*domain.Reservation)
	return reservation, args.Error(1)
}

func (m *ReservationService) ReturnReservation(
	ctx context.Context,
	reservationUID uuid.UUID,
	date domain.Date,
	inQueue bool,
) (*domain.Reservation, error) {
	args := m.Called(ctx, reservationUID, date, inQueue)
	reservation, _ := args.Get(0).(*domain.Reservation)
	return reservation, args.Error(1)
}

type RatingService struct {
	mock.Mock
}

var _ usecase.RatingService = (*RatingService)(nil)

func (m *RatingService) GetRating(ctx context.Context) (*domain.Rating, error) {
	args := m.Called(ctx)
	rating, _ := args.Get(0).(*domain.Rating)
	return rating, args.Error(1)
}

func (m *RatingService) CloseReservation(
	ctx context.Context,
	closed domain.ClosedReservation,
	inQueue bool,
) (*domain.RatingChange, error) {
	args := m.Called(ctx, closed, inQueue)
	change, _ := args.Get(0).(*domain.RatingChange)
	return change, args.Error(1)
}
