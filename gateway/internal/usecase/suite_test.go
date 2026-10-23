package usecase_test

import (
	"io"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase/mocks"
)

func TestMain(m *testing.M) {
	_ = logger.Init(logger.Settings{Level: "error", Format: logger.FormatJSON, Service: "test", Output: io.Discard})
	os.Exit(m.Run())
}

const username = "Test Max"

var (
	libraryUID     = uuid.MustParse("83575e12-7ce0-48ee-9931-51919ff3c9ee")
	bookUID        = uuid.MustParse("f7cdc58f-2caf-4b15-9727-f89dcc629b27")
	reservationUID = uuid.MustParse("f464ca3a-fcf7-4e3f-86f0-76c7bba96f72")
)

type suite struct {
	libraries    *mocks.LibraryService
	reservations *mocks.ReservationService
	ratings      *mocks.RatingService
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{
		libraries:    &mocks.LibraryService{},
		reservations: &mocks.ReservationService{},
		ratings:      &mocks.RatingService{},
	}

	t.Cleanup(func() {
		s.libraries.AssertExpectations(t)
		s.reservations.AssertExpectations(t)
		s.ratings.AssertExpectations(t)
	})
	return s
}

func (s *suite) reservationUseCase() *usecase.ReservationUseCase {
	return usecase.NewReservationUseCase(s.reservations, s.libraries, s.ratings)
}

func theBook() domain.Book {
	return domain.Book{
		BookUID: bookUID,
		Name:    "Краткий курс C++ в 7 томах",
		Author:  "Бьерн Страуструп",
		Genre:   "Научная фантастика",
	}
}

func theLibrary() domain.Library {
	return domain.Library{
		LibraryUID: libraryUID,
		Name:       "Библиотека имени 7 Непьющих",
		City:       "Москва",
		Address:    "2-я Бауманская ул., д.5, стр.1",
	}
}

func date(raw string) domain.Date {
	parsed, err := domain.ParseDate(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}

func rentedReservation() domain.Reservation {
	return domain.Reservation{
		ReservationUID:  reservationUID,
		BookUID:         bookUID,
		LibraryUID:      libraryUID,
		Status:          domain.StatusRented,
		StartDate:       date("2021-10-09"),
		TillDate:        date("2021-10-11"),
		ConditionAtRent: domain.ConditionExcellent,
	}
}
