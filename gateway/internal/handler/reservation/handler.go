package reservation

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/common"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/dto"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type ReservationHandler struct {
	reservations *usecase.ReservationUseCase
	validate     *validator.Validate
	errors       common.ErrorWriter
}

func NewReservationHandler(reservations *usecase.ReservationUseCase) *ReservationHandler {
	return &ReservationHandler{
		reservations: reservations,
		validate:     common.NewValidator(),
		errors:       common.NewErrorWriter("handler.reservation"),
	}
}

func (h *ReservationHandler) Register(r chi.Router) {
	r.Route("/api/v1/reservations", func(r chi.Router) {
		r.Get("/", h.listReservations)
		r.Post("/", h.takeBook)
		r.Post("/{reservationUid}/return", h.returnBook)
	})
}

func (h *ReservationHandler) listReservations(w http.ResponseWriter, r *http.Request) {
	username, ok := common.Username(w, r)
	if !ok {
		return
	}

	reservations, err := h.reservations.ListReservations(r.Context(), username)
	if err != nil {
		h.errors.Write(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewBookReservationResponses(reservations))
}

func (h *ReservationHandler) takeBook(w http.ResponseWriter, r *http.Request) {
	username, ok := common.Username(w, r)
	if !ok {
		return
	}

	var request dto.TakeBookRequest
	if !h.decode(w, r, &request) {
		return
	}

	command, err := takeBookCommand(request)
	if err != nil {
		common.BadRequest(w, err)
		return
	}

	taken, err := h.reservations.TakeBook(r.Context(), username, command)
	if err != nil {
		h.errors.Write(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewTakeBookResponse(*taken))
}

func (h *ReservationHandler) returnBook(w http.ResponseWriter, r *http.Request) {
	username, ok := common.Username(w, r)
	if !ok {
		return
	}

	reservationUID, ok := common.PathUUID(w, r, "reservationUid")
	if !ok {
		return
	}

	var request dto.ReturnBookRequest
	if !h.decode(w, r, &request) {
		return
	}

	date, err := domain.ParseDate(request.Date)
	if err != nil {
		common.BadParam(w, "date", err)
		return
	}

	err = h.reservations.ReturnBook(r.Context(), username, usecase.ReturnBookCommand{
		ReservationUID: reservationUID,
		Condition:      domain.BookCondition(request.Condition),
		Date:           date,
	})
	if err != nil {
		h.errors.Write(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ReservationHandler) decode(w http.ResponseWriter, r *http.Request, request any) bool {
	if err := common.DecodeJSON(r, request); err != nil {
		common.BadRequest(w, err)
		return false
	}
	if err := h.validate.Struct(request); err != nil {
		common.BadRequest(w, err)
		return false
	}
	return true
}

func takeBookCommand(request dto.TakeBookRequest) (usecase.TakeBookCommand, error) {
	bookUID, err := uuid.Parse(request.BookUID)
	if err != nil {
		return usecase.TakeBookCommand{}, err
	}

	libraryUID, err := uuid.Parse(request.LibraryUID)
	if err != nil {
		return usecase.TakeBookCommand{}, err
	}

	tillDate, err := domain.ParseDate(request.TillDate)
	if err != nil {
		return usecase.TakeBookCommand{}, err
	}

	return usecase.TakeBookCommand{BookUID: bookUID, LibraryUID: libraryUID, TillDate: tillDate}, nil
}
