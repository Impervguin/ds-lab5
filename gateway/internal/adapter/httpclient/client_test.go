package httpclient_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/adapter/httpclient"
	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
)

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
)

type upstream struct {
	method  string
	target  string
	headers http.Header
	body    map[string]any
	status  int
	answer  string
}

func newUpstream(t *testing.T, answer string) (*upstream, string) {
	t.Helper()

	recorded := &upstream{status: http.StatusOK, answer: answer}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.method = r.Method
		recorded.target = r.URL.RequestURI()
		recorded.headers = r.Header.Clone()

		if raw, err := io.ReadAll(r.Body); err == nil && len(raw) > 0 {
			require.NoError(t, json.Unmarshal(raw, &recorded.body))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(recorded.status)
		_, _ = io.WriteString(w, recorded.answer)
	}))
	t.Cleanup(server.Close)

	return recorded, server.URL
}

func failingUpstream(t *testing.T, status int) string {
	t.Helper()

	recorded, url := newUpstream(t, `{"message":"nope"}`)
	recorded.status = status
	return url
}

func TestLibraryClientListsLibrariesOfACity(t *testing.T) {
	recorded, url := newUpstream(t, `{"page":1,"pageSize":10,"totalElements":1,"items":[
		{"libraryUid":"`+libraryUIDValue+`","name":"Библиотека имени 7 Непьющих","address":"2-я Бауманская ул., д.5, стр.1","city":"Москва"}]}`)

	page, err := httpclient.NewLibraryClient(httpclient.NewBaseClient(url), nil).ListLibraries(context.Background(), "Москва", 1, 10)

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, recorded.method)
	assert.Equal(t, "/api/v1/libraries?city=%D0%9C%D0%BE%D1%81%D0%BA%D0%B2%D0%B0&page=1&size=10", recorded.target)
	assert.Equal(t, 1, page.TotalElements)
	require.Len(t, page.Items, 1)
	assert.Equal(t, libraryUID, page.Items[0].LibraryUID)
	assert.Equal(t, "Москва", page.Items[0].City)
}

func TestLibraryClientListsBooksOfALibrary(t *testing.T) {
	recorded, url := newUpstream(t, `{"page":1,"pageSize":25,"totalElements":1,"items":[
		{"bookUid":"`+bookUIDValue+`","name":"Краткий курс C++ в 7 томах","author":"Бьерн Страуструп","genre":"Научная фантастика","condition":"EXCELLENT","availableCount":1}]}`)

	page, err := httpclient.NewLibraryClient(httpclient.NewBaseClient(url), nil).ListBooks(context.Background(), libraryUID, 1, 25, true)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/libraries/"+libraryUIDValue+"/books?page=1&showAll=true&size=25", recorded.target)
	require.Len(t, page.Items, 1)
	assert.Equal(t, domain.ConditionExcellent, page.Items[0].Condition)
	assert.Equal(t, 1, page.Items[0].AvailableCount)
	assert.Equal(t, bookUID, page.Items[0].Book.BookUID)
}

func TestLibraryClientTakesTheBestCopy(t *testing.T) {
	recorded, url := newUpstream(t, `{"condition":"GOOD","availableCount":0}`)

	taken, err := httpclient.NewLibraryClient(httpclient.NewBaseClient(url), nil).TakeBook(context.Background(), libraryUID, bookUID, false)

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, recorded.method)
	assert.Equal(t, "/api/v1/libraries/"+libraryUIDValue+"/books/"+bookUIDValue+"/take", recorded.target)
	assert.Equal(t, domain.BookCopy{Condition: domain.ConditionGood, AvailableCount: 0}, *taken)
}

func TestLibraryClientReportsThatNoCopiesAreLeft(t *testing.T) {
	url := failingUpstream(t, http.StatusConflict)

	_, err := httpclient.NewLibraryClient(httpclient.NewBaseClient(url), nil).TakeBook(context.Background(), libraryUID, bookUID, false)

	assert.ErrorIs(t, err, domain.ErrNoAvailableCopies)
}

func TestLibraryClientShelvesTheCopyInItsNewCondition(t *testing.T) {
	recorded, url := newUpstream(t, `{"condition":"BAD","availableCount":1}`)

	shelved, err := httpclient.NewLibraryClient(httpclient.NewBaseClient(url), nil).ReturnBook(context.Background(), libraryUID, bookUID, domain.ConditionBad, false)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/libraries/"+libraryUIDValue+"/books/"+bookUIDValue+"/return", recorded.target)
	assert.Equal(t, map[string]any{"condition": "BAD"}, recorded.body)
	assert.Equal(t, 1, shelved.AvailableCount)
}

func TestLibraryClientReportsAnUnknownLibrary(t *testing.T) {
	url := failingUpstream(t, http.StatusNotFound)

	_, err := httpclient.NewLibraryClient(httpclient.NewBaseClient(url), nil).GetLibrary(context.Background(), libraryUID)

	assert.ErrorIs(t, err, domain.ErrLibraryNotFound)
}

func TestLibraryClientReportsAnUnknownBook(t *testing.T) {
	url := failingUpstream(t, http.StatusNotFound)

	_, err := httpclient.NewLibraryClient(httpclient.NewBaseClient(url), nil).GetBook(context.Background(), bookUID)

	assert.ErrorIs(t, err, domain.ErrBookNotFound)
}

func TestReservationClientCountsTheBooksOnHands(t *testing.T) {
	recorded, url := newUpstream(t, `{"count":3}`)

	count, err := httpclient.NewReservationClient(httpclient.NewBaseClient(url), nil).CountReservations(context.Background(), username, domain.StatusRented)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/reservations/count?status=RENTED", recorded.target)
	assert.Equal(t, username, recorded.headers.Get("X-User-Name"))
	assert.Equal(t, 3, count)
}

func TestReservationClientListsEveryReservationOfTheUser(t *testing.T) {
	recorded, url := newUpstream(t, `[{"reservationUid":"`+reservationUIDVal+`","bookUid":"`+bookUIDValue+`",
		"libraryUid":"`+libraryUIDValue+`","status":"RENTED","startDate":"2021-10-09","tillDate":"2021-10-11",
		"conditionAtRent":"EXCELLENT"}]`)

	reservations, err := httpclient.NewReservationClient(httpclient.NewBaseClient(url), nil).ListReservations(context.Background(), username, "")

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/reservations", recorded.target)
	require.Len(t, reservations, 1)
	assert.Equal(t, reservationUID, reservations[0].ReservationUID)
	assert.Equal(t, "2021-10-09", reservations[0].StartDate.String())
	assert.Equal(t, domain.ConditionExcellent, reservations[0].ConditionAtRent)
}

func TestReservationClientCreatesAReservationWithTheIssuedCondition(t *testing.T) {
	recorded, url := newUpstream(t, `{"reservationUid":"`+reservationUIDVal+`","bookUid":"`+bookUIDValue+`",
		"libraryUid":"`+libraryUIDValue+`","status":"RENTED","startDate":"2021-10-09","tillDate":"2021-10-11",
		"conditionAtRent":"GOOD"}`)

	tillDate, err := domain.ParseDate("2021-10-11")
	require.NoError(t, err)

	reservation, err := httpclient.NewReservationClient(httpclient.NewBaseClient(url), nil).CreateReservation(context.Background(), username, domain.NewReservation{
		BookUID:         bookUID,
		LibraryUID:      libraryUID,
		TillDate:        tillDate,
		ConditionAtRent: domain.ConditionGood,
	}, false)

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, recorded.method)
	assert.Equal(t, "/api/v1/reservations", recorded.target)
	assert.Equal(t, map[string]any{
		"bookUid":         bookUIDValue,
		"libraryUid":      libraryUIDValue,
		"tillDate":        "2021-10-11",
		"conditionAtRent": "GOOD",
	}, recorded.body)
	assert.Equal(t, domain.StatusRented, reservation.Status)
}

func TestReservationClientClosesAReservation(t *testing.T) {
	recorded, url := newUpstream(t, `{"reservationUid":"`+reservationUIDVal+`","bookUid":"`+bookUIDValue+`",
		"libraryUid":"`+libraryUIDValue+`","status":"EXPIRED","startDate":"2021-10-09","tillDate":"2021-10-11",
		"conditionAtRent":"EXCELLENT"}`)

	date, err := domain.ParseDate("2021-10-15")
	require.NoError(t, err)

	reservation, err := httpclient.NewReservationClient(httpclient.NewBaseClient(url), nil).ReturnReservation(context.Background(), username, reservationUID, date, false)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/reservations/"+reservationUIDVal+"/return", recorded.target)
	assert.Equal(t, map[string]any{"date": "2021-10-15"}, recorded.body)
	assert.Equal(t, domain.StatusExpired, reservation.Status)
}

func TestReservationClientReportsAnUnknownReservation(t *testing.T) {
	url := failingUpstream(t, http.StatusNotFound)
	date, err := domain.ParseDate("2021-10-15")
	require.NoError(t, err)

	_, err = httpclient.NewReservationClient(httpclient.NewBaseClient(url), nil).ReturnReservation(context.Background(), username, reservationUID, date, false)

	assert.ErrorIs(t, err, domain.ErrReservationNotFound)
}

func TestReservationClientReportsAClosedReservation(t *testing.T) {
	url := failingUpstream(t, http.StatusConflict)
	date, err := domain.ParseDate("2021-10-15")
	require.NoError(t, err)

	_, err = httpclient.NewReservationClient(httpclient.NewBaseClient(url), nil).ReturnReservation(context.Background(), username, reservationUID, date, false)

	assert.ErrorIs(t, err, domain.ErrReservationClosed)
}

func TestRatingClientReadsStarsAndTheBookLimit(t *testing.T) {
	recorded, url := newUpstream(t, `{"stars":75,"maxBooks":25}`)

	rating, err := httpclient.NewRatingClient(httpclient.NewBaseClient(url), nil).GetRating(context.Background(), username)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/rating", recorded.target)
	assert.Equal(t, username, recorded.headers.Get("X-User-Name"))
	assert.Equal(t, domain.Rating{Stars: 75, MaxBooks: 25}, *rating)
}

func TestRatingClientSendsTheFactsOfAClosedReservation(t *testing.T) {
	recorded, url := newUpstream(t, `{"delta":-20,"stars":55}`)

	change, err := httpclient.NewRatingClient(httpclient.NewBaseClient(url), nil).CloseReservation(context.Background(), username, domain.ClosedReservation{
		ReservationUID:    reservationUID,
		Status:            domain.StatusExpired,
		ConditionAtRent:   domain.ConditionExcellent,
		ConditionOnReturn: domain.ConditionBad,
	}, false)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/rating/reservation-closed", recorded.target)
	assert.Equal(t, map[string]any{
		"reservationUid":    reservationUIDVal,
		"reservationStatus": "EXPIRED",
		"conditionAtRent":   "EXCELLENT",
		"conditionOnReturn": "BAD",
	}, recorded.body)
	assert.Equal(t, domain.RatingChange{Delta: -20, Stars: 55}, *change)
}

func TestClientReportsAnUnexpectedStatus(t *testing.T) {
	url := failingUpstream(t, http.StatusInternalServerError)

	_, err := httpclient.NewRatingClient(httpclient.NewBaseClient(url), nil).GetRating(context.Background(), username)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
}
