package common

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type ErrorWriter struct {
	log *slog.Logger
}

func NewErrorWriter(name string) ErrorWriter {
	return ErrorWriter{log: logger.Named(name)}
}

func (e ErrorWriter) Write(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrLibraryNotFound):
		WriteError(w, http.StatusNotFound, "Library not found")
	case errors.Is(err, domain.ErrBookNotFound):
		WriteError(w, http.StatusNotFound, "Book not found")
	case errors.Is(err, domain.ErrReservationNotFound):
		WriteError(w, http.StatusNotFound, "Reservation not found")
	case errors.Is(err, domain.ErrNoAvailableCopies):
		WriteError(w, http.StatusConflict, "No available copies of the book left")
	case errors.Is(err, domain.ErrReservationClosed):
		WriteError(w, http.StatusConflict, "Reservation is already closed")
	case errors.Is(err, domain.ErrBookLimitReached):
		WriteValidationError(w, "The number of books taken has reached the limit", nil)
	case errors.Is(err, usecase.ErrLibraryUnavailable):
		WriteError(w, http.StatusServiceUnavailable, "Library Service unavailable")
	case errors.Is(err, usecase.ErrReservationUnavailable):
		WriteError(w, http.StatusServiceUnavailable, "Reservation Service unavailable")
	case errors.Is(err, usecase.ErrRatingUnavailable):
		WriteError(w, http.StatusServiceUnavailable, "Service unavailable")
	case errors.Is(err, usecase.ErrServiceUnavailable):
		WriteError(w, http.StatusServiceUnavailable, "Service unavailable")
	default:
		e.log.ErrorContext(r.Context(), "request failed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
		)
		WriteError(w, http.StatusInternalServerError, "Internal server error")
	}
}
