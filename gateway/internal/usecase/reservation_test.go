package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

var errService = errors.New("service is unavailable")

func takeCommand() usecase.TakeBookCommand {
	return usecase.TakeBookCommand{BookUID: bookUID, LibraryUID: libraryUID, TillDate: date("2021-10-11")}
}

func TestTakeBookCollectsReservationBookLibraryAndRating(t *testing.T) {
	s := newSuite(t)
	reservation := rentedReservation()

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(1, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).
		Return(&domain.BookCopy{Condition: domain.ConditionExcellent, AvailableCount: 0}, nil)
	s.reservations.On("CreateReservation", mock.Anything, username, domain.NewReservation{
		BookUID:         bookUID,
		LibraryUID:      libraryUID,
		TillDate:        date("2021-10-11"),
		ConditionAtRent: domain.ConditionExcellent,
	}, false).Return(&reservation, nil)
	book := theBook()
	s.libraries.On("GetBook", mock.Anything, bookUID).Return(&book, nil)
	library := theLibrary()
	s.libraries.On("GetLibrary", mock.Anything, libraryUID).Return(&library, nil)

	taken, err := s.reservationUseCase().TakeBook(context.Background(), username, takeCommand())

	require.NoError(t, err)
	assert.Equal(t, reservation, taken.Reservation)
	assert.Equal(t, book, taken.Book)
	assert.Equal(t, library, taken.Library)
	assert.Equal(t, domain.Rating{Stars: 75, MaxBooks: 25}, taken.Rating)
}

func TestTakeBookPassesTheIssuedConditionToTheReservation(t *testing.T) {
	s := newSuite(t)
	reservation := rentedReservation()

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).
		Return(&domain.BookCopy{Condition: domain.ConditionBad, AvailableCount: 0}, nil)
	s.reservations.On("CreateReservation", mock.Anything, username, mock.Anything, false).
		Run(func(args mock.Arguments) {
			request, ok := args.Get(2).(domain.NewReservation)
			require.True(t, ok, "CreateReservation was called without a request")
			assert.Equal(t, domain.ConditionBad, request.ConditionAtRent)
		}).
		Return(&reservation, nil)
	book := theBook()
	s.libraries.On("GetBook", mock.Anything, bookUID).Return(&book, nil)
	library := theLibrary()
	s.libraries.On("GetLibrary", mock.Anything, libraryUID).Return(&library, nil)

	_, err := s.reservationUseCase().TakeBook(context.Background(), username, takeCommand())

	require.NoError(t, err)
}

func TestTakeBookStopsWhenTheBookLimitIsReached(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(2, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 6, MaxBooks: 2}, nil)

	_, err := s.reservationUseCase().TakeBook(context.Background(), username, takeCommand())

	assert.ErrorIs(t, err, domain.ErrBookLimitReached)
	s.libraries.AssertNotCalled(t, "TakeBook", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTakeBookDoesNotReserveWhenNoCopiesAreLeft(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).Return(nil, domain.ErrNoAvailableCopies)

	_, err := s.reservationUseCase().TakeBook(context.Background(), username, takeCommand())

	assert.ErrorIs(t, err, domain.ErrNoAvailableCopies)
	s.reservations.AssertNotCalled(t, "CreateReservation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTakeBookGivesTheCopyBackWhenTheReservationFails(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).
		Return(&domain.BookCopy{Condition: domain.ConditionGood, AvailableCount: 0}, nil)
	s.reservations.On("CreateReservation", mock.Anything, username, mock.Anything, false).Return(nil, errService)
	s.libraries.On("ReturnBook", mock.Anything, libraryUID, bookUID, domain.ConditionGood, true).
		Return(&domain.BookCopy{Condition: domain.ConditionGood, AvailableCount: 1}, nil)

	_, err := s.reservationUseCase().TakeBook(context.Background(), username, takeCommand())

	assert.ErrorIs(t, err, errService)
}

func TestTakeBookSurvivesAFailedCompensation(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).
		Return(&domain.BookCopy{Condition: domain.ConditionGood, AvailableCount: 0}, nil)
	s.reservations.On("CreateReservation", mock.Anything, username, mock.Anything, false).Return(nil, errService)
	s.libraries.On("ReturnBook", mock.Anything, libraryUID, bookUID, domain.ConditionGood, true).Return(nil, errService)

	_, err := s.reservationUseCase().TakeBook(context.Background(), username, takeCommand())

	assert.ErrorIs(t, err, errService)
}

func TestListReservationsEnrichesEveryReservation(t *testing.T) {
	s := newSuite(t)
	reservation := rentedReservation()

	s.reservations.On("ListReservations", mock.Anything, username, domain.ReservationStatus("")).
		Return([]domain.Reservation{reservation}, nil)
	book := theBook()
	s.libraries.On("GetBook", mock.Anything, bookUID).Return(&book, nil)
	library := theLibrary()
	s.libraries.On("GetLibrary", mock.Anything, libraryUID).Return(&library, nil)

	details, err := s.reservationUseCase().ListReservations(context.Background(), username)

	require.NoError(t, err)
	require.Len(t, details, 1)
	assert.Equal(t, reservation, details[0].Reservation)
	assert.Equal(t, book, details[0].Book)
	assert.Equal(t, library, details[0].Library)
}

func TestListReservationsAsksAboutEveryBookOnlyOnce(t *testing.T) {
	s := newSuite(t)

	closed := rentedReservation()
	closed.ReservationUID = uuid.MustParse("5ae1d0f4-0a6e-4bd1-9e5b-27b17c0a1b57")
	closed.Status = domain.StatusReturned

	s.reservations.On("ListReservations", mock.Anything, username, domain.ReservationStatus("")).
		Return([]domain.Reservation{rentedReservation(), closed}, nil)
	book := theBook()
	s.libraries.On("GetBook", mock.Anything, bookUID).Return(&book, nil).Once()
	library := theLibrary()
	s.libraries.On("GetLibrary", mock.Anything, libraryUID).Return(&library, nil).Once()

	details, err := s.reservationUseCase().ListReservations(context.Background(), username)

	require.NoError(t, err)
	assert.Len(t, details, 2)
}

func TestListReservationsReportsAnUnavailableLibrary(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("ListReservations", mock.Anything, username, domain.ReservationStatus("")).
		Return([]domain.Reservation{rentedReservation()}, nil)
	s.libraries.On("GetBook", mock.Anything, bookUID).Return(nil, errService)

	_, err := s.reservationUseCase().ListReservations(context.Background(), username)

	assert.ErrorIs(t, err, errService)
}

func TestReturnBookShelvesTheCopyAndReportsTheFactsToRating(t *testing.T) {
	s := newSuite(t)

	closed := rentedReservation()
	closed.Status = domain.StatusExpired

	s.reservations.On("ReturnReservation", mock.Anything, username, reservationUID, date("2021-10-15"), false).
		Return(&closed, nil)
	s.libraries.On("ReturnBook", mock.Anything, libraryUID, bookUID, domain.ConditionBad, true).
		Return(&domain.BookCopy{Condition: domain.ConditionBad, AvailableCount: 1}, nil)
	s.ratings.On("CloseReservation", mock.Anything, username, domain.ClosedReservation{
		ReservationUID:    reservationUID,
		Status:            domain.StatusExpired,
		ConditionAtRent:   domain.ConditionExcellent,
		ConditionOnReturn: domain.ConditionBad,
	}, true).Return(&domain.RatingChange{Delta: -20, Stars: 55}, nil)

	err := s.reservationUseCase().ReturnBook(context.Background(), username, usecase.ReturnBookCommand{
		ReservationUID: reservationUID,
		Condition:      domain.ConditionBad,
		Date:           date("2021-10-15"),
	})

	require.NoError(t, err)
}

func TestReturnBookStopsWhenTheReservationIsUnknown(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("ReturnReservation", mock.Anything, username, reservationUID, date("2021-10-11"), false).
		Return(nil, domain.ErrReservationNotFound)

	err := s.reservationUseCase().ReturnBook(context.Background(), username, usecase.ReturnBookCommand{
		ReservationUID: reservationUID,
		Condition:      domain.ConditionExcellent,
		Date:           date("2021-10-11"),
	})

	assert.ErrorIs(t, err, domain.ErrReservationNotFound)
	s.libraries.AssertNotCalled(t, "ReturnBook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	s.ratings.AssertNotCalled(t, "CloseReservation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReturnBookDoesNotTouchRatingWhenTheLibraryFails(t *testing.T) {
	s := newSuite(t)
	closed := rentedReservation()
	closed.Status = domain.StatusReturned

	s.reservations.On("ReturnReservation", mock.Anything, username, reservationUID, date("2021-10-11"), false).
		Return(&closed, nil)
	s.libraries.On("ReturnBook", mock.Anything, libraryUID, bookUID, domain.ConditionExcellent, true).
		Return(nil, domain.ErrBookNotFound)

	err := s.reservationUseCase().ReturnBook(context.Background(), username, usecase.ReturnBookCommand{
		ReservationUID: reservationUID,
		Condition:      domain.ConditionExcellent,
		Date:           date("2021-10-11"),
	})

	assert.ErrorIs(t, err, domain.ErrBookNotFound)
	s.ratings.AssertNotCalled(t, "CloseReservation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReturnBookSucceedsWhenLibraryAndRatingAreQueued(t *testing.T) {
	s := newSuite(t)
	closed := rentedReservation()
	closed.Status = domain.StatusReturned

	s.reservations.On("ReturnReservation", mock.Anything, username, reservationUID, date("2021-10-11"), false).
		Return(&closed, nil)
	s.libraries.On("ReturnBook", mock.Anything, libraryUID, bookUID, domain.ConditionExcellent, true).
		Return(nil, usecase.ErrQueued)
	s.ratings.On("CloseReservation", mock.Anything, username, mock.Anything, true).
		Return(nil, usecase.ErrQueued)

	err := s.reservationUseCase().ReturnBook(context.Background(), username, usecase.ReturnBookCommand{
		ReservationUID: reservationUID,
		Condition:      domain.ConditionExcellent,
		Date:           date("2021-10-11"),
	})

	require.NoError(t, err)
	s.ratings.AssertExpectations(t)
}

func TestTakeBookQueuesTheCompensationWhenTheLibraryIsDown(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).
		Return(&domain.BookCopy{Condition: domain.ConditionGood}, nil)
	s.reservations.On("CreateReservation", mock.Anything, username, mock.Anything, false).Return(nil, errService)
	s.libraries.On("ReturnBook", mock.Anything, libraryUID, bookUID, domain.ConditionGood, true).
		Return(nil, usecase.ErrQueued)

	_, err := s.reservationUseCase().TakeBook(context.Background(), username, takeCommand())

	assert.ErrorIs(t, err, errService)
	s.libraries.AssertExpectations(t)
}
