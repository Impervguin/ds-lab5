package reservation_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/reservation"
	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase/mocks"
)

func TestMain(m *testing.M) {
	_ = logger.Init(logger.Settings{Level: "error", Format: logger.FormatJSON, Service: "test", Output: io.Discard})
	os.Exit(m.Run())
}

const (
	username          = "Test Max"
	libraryUIDValue   = "83575e12-7ce0-48ee-9931-51919ff3c9ee"
	bookUIDValue      = "f7cdc58f-2caf-4b15-9727-f89dcc629b27"
	reservationUIDVal = "f464ca3a-fcf7-4e3f-86f0-76c7bba96f72"
)

var (
	libraryUID     = uuid.MustParse(libraryUIDValue)
	bookUID        = uuid.MustParse(bookUIDValue)
	reservationUID = uuid.MustParse(reservationUIDVal)
	errService     = errors.New("service is unavailable")
)

type suite struct {
	libraries    *mocks.LibraryService
	reservations *mocks.ReservationService
	ratings      *mocks.RatingService
	routes       http.Handler
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{
		libraries:    &mocks.LibraryService{},
		reservations: &mocks.ReservationService{},
		ratings:      &mocks.RatingService{},
	}

	router := chi.NewRouter()
	reservation.NewReservationHandler(
		usecase.NewReservationUseCase(s.reservations, s.libraries, s.ratings),
	).Register(router)
	s.routes = router

	t.Cleanup(func() {
		s.libraries.AssertExpectations(t)
		s.reservations.AssertExpectations(t)
		s.ratings.AssertExpectations(t)
	})
	return s
}

func (s *suite) do(t *testing.T, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(method, target, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}

	recorder := httptest.NewRecorder()
	s.routes.ServeHTTP(recorder, request)
	return recorder
}

func asUser() map[string]string {
	return map[string]string{"X-User-Name": username}
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), target))
}

func date(t *testing.T, raw string) domain.Date {
	t.Helper()

	parsed, err := domain.ParseDate(raw)
	require.NoError(t, err)
	return parsed
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

func (s *suite) catalogueAnswers() {
	book := theBook()
	library := theLibrary()
	s.libraries.On("GetBook", mock.Anything, bookUID).Return(&book, nil)
	s.libraries.On("GetLibrary", mock.Anything, libraryUID).Return(&library, nil)
}

func rentedReservation(t *testing.T) domain.Reservation {
	t.Helper()

	return domain.Reservation{
		ReservationUID:  reservationUID,
		BookUID:         bookUID,
		LibraryUID:      libraryUID,
		Status:          domain.StatusRented,
		StartDate:       date(t, "2021-10-09"),
		TillDate:        date(t, "2021-10-11"),
		ConditionAtRent: domain.ConditionExcellent,
	}
}

func TestListReservationsAnswersWithAnArrayOfEnrichedReservations(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("ListReservations", mock.Anything, username, domain.ReservationStatus("")).
		Return([]domain.Reservation{rentedReservation(t)}, nil)
	s.catalogueAnswers()

	recorder := s.do(t, http.MethodGet, "/api/v1/reservations", "", asUser())

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")

	var response []map[string]any
	decode(t, recorder, &response)
	require.Len(t, response, 1)
	assert.Equal(t, reservationUIDVal, response[0]["reservationUid"])
	assert.Equal(t, "RENTED", response[0]["status"])
	assert.Equal(t, "2021-10-09", response[0]["startDate"])
	assert.Equal(t, "2021-10-11", response[0]["tillDate"])
	assert.Equal(t, bookUIDValue, response[0]["book"].(map[string]any)["bookUid"])
	assert.Equal(t, libraryUIDValue, response[0]["library"].(map[string]any)["libraryUid"])
}

func TestListReservationsAnswersWithAnEmptyArray(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("ListReservations", mock.Anything, username, domain.ReservationStatus("")).
		Return([]domain.Reservation{}, nil)

	recorder := s.do(t, http.MethodGet, "/api/v1/reservations", "", asUser())

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, "[]", recorder.Body.String())
}

func TestListReservationsDemandsTheUserName(t *testing.T) {
	s := newSuite(t)

	recorder := s.do(t, http.MethodGet, "/api/v1/reservations", "", nil)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "X-User-Name")
}

func TestTakeBookAnswersWithTheReservationBookLibraryAndRating(t *testing.T) {
	s := newSuite(t)
	reserved := rentedReservation(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).
		Return(&domain.BookCopy{Condition: domain.ConditionExcellent, AvailableCount: 0}, nil)
	s.reservations.On("CreateReservation", mock.Anything, username, domain.NewReservation{
		BookUID:         bookUID,
		LibraryUID:      libraryUID,
		TillDate:        date(t, "2021-10-11"),
		ConditionAtRent: domain.ConditionExcellent,
	}, false).Return(&reserved, nil)
	s.catalogueAnswers()

	body := `{"bookUid":"` + bookUIDValue + `","libraryUid":"` + libraryUIDValue + `","tillDate":"2021-10-11"}`
	recorder := s.do(t, http.MethodPost, "/api/v1/reservations", body, asUser())

	require.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]any
	decode(t, recorder, &response)
	assert.Equal(t, reservationUIDVal, response["reservationUid"])
	assert.Equal(t, "RENTED", response["status"])
	assert.Equal(t, "2021-10-11", response["tillDate"])
	assert.Equal(t, "Краткий курс C++ в 7 томах", response["book"].(map[string]any)["name"])
	assert.Equal(t, "Москва", response["library"].(map[string]any)["city"])
	assert.Equal(t, float64(75), response["rating"].(map[string]any)["stars"])
}

func TestTakeBookAnswersWithFourHundredWhenTheLimitIsReached(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(25, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)

	body := `{"bookUid":"` + bookUIDValue + `","libraryUid":"` + libraryUIDValue + `","tillDate":"2021-10-11"}`
	recorder := s.do(t, http.MethodPost, "/api/v1/reservations", body, asUser())

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestTakeBookAnswersWithConflictWhenNoCopiesAreLeft(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, nil)
	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)
	s.libraries.On("TakeBook", mock.Anything, libraryUID, bookUID, false).Return(nil, domain.ErrNoAvailableCopies)

	body := `{"bookUid":"` + bookUIDValue + `","libraryUid":"` + libraryUIDValue + `","tillDate":"2021-10-11"}`
	recorder := s.do(t, http.MethodPost, "/api/v1/reservations", body, asUser())

	assert.Equal(t, http.StatusConflict, recorder.Code)
}

func TestTakeBookRejectsAMalformedRequest(t *testing.T) {
	s := newSuite(t)

	cases := map[string]string{
		"not a uuid":    `{"bookUid":"not-a-uuid","libraryUid":"` + libraryUIDValue + `","tillDate":"2021-10-11"}`,
		"not a date":    `{"bookUid":"` + bookUIDValue + `","libraryUid":"` + libraryUIDValue + `","tillDate":"11.10.2021"}`,
		"missing field": `{"bookUid":"` + bookUIDValue + `","tillDate":"2021-10-11"}`,
		"unknown field": `{"bookUid":"` + bookUIDValue + `","libraryUid":"` + libraryUIDValue + `","tillDate":"2021-10-11","stars":5}`,
		"not even json": `{`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			recorder := s.do(t, http.MethodPost, "/api/v1/reservations", body, asUser())
			assert.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func TestTakeBookAnswersWithFiveHundredWhenAServiceIsDown(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("CountReservations", mock.Anything, username, domain.StatusRented).Return(0, errService)

	body := `{"bookUid":"` + bookUIDValue + `","libraryUid":"` + libraryUIDValue + `","tillDate":"2021-10-11"}`
	recorder := s.do(t, http.MethodPost, "/api/v1/reservations", body, asUser())

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestReturnBookAnswersWithNoContent(t *testing.T) {
	s := newSuite(t)
	closed := rentedReservation(t)
	closed.Status = domain.StatusReturned

	s.reservations.On("ReturnReservation", mock.Anything, username, reservationUID, date(t, "2021-10-11"), false).
		Return(&closed, nil)
	s.libraries.On("ReturnBook", mock.Anything, libraryUID, bookUID, domain.ConditionExcellent, true).
		Return(&domain.BookCopy{Condition: domain.ConditionExcellent, AvailableCount: 1}, nil)
	s.ratings.On("CloseReservation", mock.Anything, username, domain.ClosedReservation{
		ReservationUID:    reservationUID,
		Status:            domain.StatusReturned,
		ConditionAtRent:   domain.ConditionExcellent,
		ConditionOnReturn: domain.ConditionExcellent,
	}, true).Return(&domain.RatingChange{Delta: 1, Stars: 76}, nil)

	recorder := s.do(t, http.MethodPost,
		"/api/v1/reservations/"+reservationUIDVal+"/return",
		`{"condition":"EXCELLENT","date":"2021-10-11"}`, asUser())

	require.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func TestReturnBookAnswersWithNotFoundForAnUnknownReservation(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("ReturnReservation", mock.Anything, username, reservationUID, date(t, "2021-10-11"), false).
		Return(nil, domain.ErrReservationNotFound)

	recorder := s.do(t, http.MethodPost,
		"/api/v1/reservations/"+reservationUIDVal+"/return",
		`{"condition":"EXCELLENT","date":"2021-10-11"}`, asUser())

	require.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Reservation not found")
}

func TestReturnBookAnswersWithConflictForAClosedReservation(t *testing.T) {
	s := newSuite(t)

	s.reservations.On("ReturnReservation", mock.Anything, username, reservationUID, date(t, "2021-10-11"), false).
		Return(nil, domain.ErrReservationClosed)

	recorder := s.do(t, http.MethodPost,
		"/api/v1/reservations/"+reservationUIDVal+"/return",
		`{"condition":"EXCELLENT","date":"2021-10-11"}`, asUser())

	assert.Equal(t, http.StatusConflict, recorder.Code)
}

func TestReturnBookRejectsAnUnknownCondition(t *testing.T) {
	s := newSuite(t)

	recorder := s.do(t, http.MethodPost,
		"/api/v1/reservations/"+reservationUIDVal+"/return",
		`{"condition":"BURNED","date":"2021-10-11"}`, asUser())

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "condition")
}

func TestReturnBookRejectsAMalformedReservationUID(t *testing.T) {
	s := newSuite(t)

	recorder := s.do(t, http.MethodPost, "/api/v1/reservations/not-a-uuid/return",
		`{"condition":"EXCELLENT","date":"2021-10-11"}`, asUser())

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "reservationUid")
}

func TestReturnBookDemandsTheUserName(t *testing.T) {
	s := newSuite(t)

	recorder := s.do(t, http.MethodPost,
		"/api/v1/reservations/"+reservationUIDVal+"/return",
		`{"condition":"EXCELLENT","date":"2021-10-11"}`, nil)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
