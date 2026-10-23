package httpclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type ReservationClient struct {
	transport HttpDoer
	retryer   *HttpRetryer
}

var _ usecase.ReservationService = (*ReservationClient)(nil)

func NewReservationClient(transport HttpDoer, retryer *HttpRetryer) *ReservationClient {
	return &ReservationClient{transport: transport, retryer: retryer}
}

func (c *ReservationClient) ListReservations(
	ctx context.Context,
	username string,
	status domain.ReservationStatus,
) ([]domain.Reservation, error) {
	var response []reservationResponse
	if err := c.transport.do(ctx, call{
		method:   http.MethodGet,
		path:     "/api/v1/reservations",
		query:    statusQuery(status),
		username: username,
		out:      &response,
	}); err != nil {
		return nil, err
	}

	reservations := make([]domain.Reservation, 0, len(response))
	for _, reservation := range response {
		reservations = append(reservations, reservation.toDomain())
	}
	return reservations, nil
}

func (c *ReservationClient) CountReservations(
	ctx context.Context,
	username string,
	status domain.ReservationStatus,
) (int, error) {
	var response struct {
		Count int `json:"count"`
	}
	if err := c.transport.do(ctx, call{
		method:   http.MethodGet,
		path:     "/api/v1/reservations/count",
		query:    statusQuery(status),
		username: username,
		out:      &response,
	}); err != nil {
		return 0, err
	}
	return response.Count, nil
}

func (c *ReservationClient) CreateReservation(
	ctx context.Context,
	username string,
	request domain.NewReservation,
	inQueue bool,
) (*domain.Reservation, error) {
	var response reservationResponse
	if err := doOrQueue(ctx, c.transport, c.retryer, inQueue, call{
		method:   http.MethodPost,
		path:     "/api/v1/reservations",
		username: username,
		body: createReservationRequest{
			BookUID:         request.BookUID,
			LibraryUID:      request.LibraryUID,
			TillDate:        request.TillDate,
			ConditionAtRent: string(request.ConditionAtRent),
		},
		out: &response,
	}); err != nil {
		return nil, err
	}

	reservation := response.toDomain()
	return &reservation, nil
}

func (c *ReservationClient) ReturnReservation(
	ctx context.Context,
	username string,
	reservationUID uuid.UUID,
	date domain.Date,
	inQueue bool,
) (*domain.Reservation, error) {
	var response reservationResponse
	if err := doOrQueue(ctx, c.transport, c.retryer, inQueue, call{
		method:   http.MethodPost,
		path:     "/api/v1/reservations/" + reservationUID.String() + "/return",
		username: username,
		body:     returnReservationRequest{Date: date},
		out:      &response,
		statuses: map[int]error{
			http.StatusNotFound: domain.ErrReservationNotFound,
			http.StatusConflict: domain.ErrReservationClosed,
		},
	}); err != nil {
		return nil, err
	}

	reservation := response.toDomain()
	return &reservation, nil
}

func statusQuery(status domain.ReservationStatus) url.Values {
	if status == "" {
		return nil
	}
	return url.Values{"status": {string(status)}}
}

type reservationResponse struct {
	ReservationUID  uuid.UUID   `json:"reservationUid"`
	BookUID         uuid.UUID   `json:"bookUid"`
	LibraryUID      uuid.UUID   `json:"libraryUid"`
	Status          string      `json:"status"`
	StartDate       domain.Date `json:"startDate"`
	TillDate        domain.Date `json:"tillDate"`
	ConditionAtRent string      `json:"conditionAtRent"`
}

func (r reservationResponse) toDomain() domain.Reservation {
	return domain.Reservation{
		ReservationUID:  r.ReservationUID,
		BookUID:         r.BookUID,
		LibraryUID:      r.LibraryUID,
		Status:          domain.ReservationStatus(r.Status),
		StartDate:       r.StartDate,
		TillDate:        r.TillDate,
		ConditionAtRent: domain.BookCondition(r.ConditionAtRent),
	}
}

type createReservationRequest struct {
	BookUID         uuid.UUID   `json:"bookUid"`
	LibraryUID      uuid.UUID   `json:"libraryUid"`
	TillDate        domain.Date `json:"tillDate"`
	ConditionAtRent string      `json:"conditionAtRent"`
}

type returnReservationRequest struct {
	Date domain.Date `json:"date"`
}
