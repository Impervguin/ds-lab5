package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
)

type TakeBookCommand struct {
	BookUID    uuid.UUID
	LibraryUID uuid.UUID
	TillDate   domain.Date
}

type ReturnBookCommand struct {
	ReservationUID uuid.UUID
	Condition      domain.BookCondition
	Date           domain.Date
}

type ReservationUseCase struct {
	reservations ReservationService
	libraries    LibraryService
	ratings      RatingService
	log          *slog.Logger
}

func NewReservationUseCase(
	reservations ReservationService,
	libraries LibraryService,
	ratings RatingService,
) *ReservationUseCase {
	return &ReservationUseCase{
		reservations: reservations,
		libraries:    libraries,
		ratings:      ratings,
		log:          logger.Named("usecase.reservation"),
	}
}

func (uc *ReservationUseCase) ListReservations(ctx context.Context, username string) ([]domain.ReservationDetails, error) {
	reservations, err := uc.reservations.ListReservations(ctx, username, "")
	if err != nil {
		return nil, unavailable(err, ErrReservationUnavailable)
	}

	catalogue := uc.newCatalogue()

	details := make([]domain.ReservationDetails, 0, len(reservations))
	for _, reservation := range reservations {
		described, err := catalogue.describe(ctx, reservation)
		if err != nil {
			return nil, err
		}
		details = append(details, *described)
	}
	return details, nil
}

func (uc *ReservationUseCase) TakeBook(
	ctx context.Context,
	username string,
	command TakeBookCommand,
) (*domain.TakenBook, error) {
	rented, err := uc.reservations.CountReservations(ctx, username, domain.StatusRented)
	if err != nil {
		return nil, unavailable(err, ErrReservationUnavailable)
	}

	rating, err := uc.ratings.GetRating(ctx, username)
	if err != nil {
		return nil, unavailable(err, ErrRatingUnavailable)
	}

	if rented >= rating.MaxBooks {
		uc.log.InfoContext(ctx, "book limit reached",
			slog.String("username", username),
			slog.Int("rented", rented),
			slog.Int("maxBooks", rating.MaxBooks),
		)
		return nil, domain.ErrBookLimitReached
	}

	taken, err := uc.libraries.TakeBook(ctx, command.LibraryUID, command.BookUID, false)
	if err != nil {
		return nil, unavailable(err, ErrLibraryUnavailable)
	}

	reservation, err := uc.reservations.CreateReservation(ctx, username, domain.NewReservation{
		BookUID:         command.BookUID,
		LibraryUID:      command.LibraryUID,
		TillDate:        command.TillDate,
		ConditionAtRent: taken.Condition,
	}, false)
	if err != nil {
		uc.giveCopyBack(ctx, command.LibraryUID, command.BookUID, taken.Condition)
		return nil, unavailable(err, ErrReservationUnavailable)
	}

	described, err := uc.newCatalogue().describe(ctx, *reservation)
	if err != nil {
		return nil, err
	}

	return &domain.TakenBook{ReservationDetails: *described, Rating: *rating}, nil
}

func (uc *ReservationUseCase) ReturnBook(ctx context.Context, username string, command ReturnBookCommand) error {
	reservation, err := uc.reservations.ReturnReservation(ctx, username, command.ReservationUID, command.Date, false)
	if err != nil {
		return unavailable(err, ErrReservationUnavailable)
	}

	log := uc.log.With(
		slog.String("username", username),
		slog.String("reservationUid", reservation.ReservationUID.String()),
	)

	_, err = uc.libraries.ReturnBook(ctx, reservation.LibraryUID, reservation.BookUID, command.Condition, true)
	switch {
	case errors.Is(err, ErrQueued):
		log.WarnContext(ctx, "library is unavailable, returning the copy later")
	case err != nil:
		return unavailable(err, ErrLibraryUnavailable)
	}

	change, err := uc.ratings.CloseReservation(ctx, username, domain.ClosedReservation{
		ReservationUID:    reservation.ReservationUID,
		Status:            reservation.Status,
		ConditionAtRent:   reservation.ConditionAtRent,
		ConditionOnReturn: command.Condition,
	}, true)
	switch {
	case errors.Is(err, ErrQueued):
		log.WarnContext(ctx, "rating is unavailable, updating it later",
			slog.String("status", string(reservation.Status)),
		)
		return nil
	case err != nil:
		return unavailable(err, ErrRatingUnavailable)
	}

	log.InfoContext(ctx, "reservation closed",
		slog.String("status", string(reservation.Status)),
		slog.Int("delta", change.Delta),
		slog.Int("stars", change.Stars),
	)
	return nil
}

func (uc *ReservationUseCase) giveCopyBack(
	ctx context.Context,
	libraryUID, bookUID uuid.UUID,
	condition domain.BookCondition,
) {
	ctx = context.WithoutCancel(ctx)

	_, err := uc.libraries.ReturnBook(ctx, libraryUID, bookUID, condition, true)
	if errors.Is(err, ErrQueued) {
		uc.log.WarnContext(ctx, "compensation queued, the copy will be returned later",
			slog.String("libraryUid", libraryUID.String()),
			slog.String("bookUid", bookUID.String()),
			slog.String("condition", string(condition)),
		)
		return
	}
	if err != nil {
		uc.log.ErrorContext(ctx, "compensation failed, the copy stays written off",
			slog.String("libraryUid", libraryUID.String()),
			slog.String("bookUid", bookUID.String()),
			slog.String("condition", string(condition)),
			slog.String("error", err.Error()),
		)
		return
	}

	uc.log.WarnContext(ctx, "copy returned after a failed reservation",
		slog.String("libraryUid", libraryUID.String()),
		slog.String("bookUid", bookUID.String()),
		slog.String("condition", string(condition)),
	)
}

func (uc *ReservationUseCase) newCatalogue() *catalogue {
	return &catalogue{
		libraries: uc.libraries,
		books:     make(map[uuid.UUID]domain.Book),
		places:    make(map[uuid.UUID]domain.Library),
	}
}

type catalogue struct {
	libraries LibraryService
	books     map[uuid.UUID]domain.Book
	places    map[uuid.UUID]domain.Library
}

func (c *catalogue) describe(ctx context.Context, reservation domain.Reservation) (*domain.ReservationDetails, error) {
	book, err := c.book(ctx, reservation.BookUID)
	if err != nil {
		return nil, err
	}

	library, err := c.library(ctx, reservation.LibraryUID)
	if err != nil {
		return nil, err
	}

	return &domain.ReservationDetails{Reservation: reservation, Book: *book, Library: *library}, nil
}

func (c *catalogue) book(ctx context.Context, bookUID uuid.UUID) (*domain.Book, error) {
	if known, ok := c.books[bookUID]; ok {
		return &known, nil
	}

	book, err := c.libraries.GetBook(ctx, bookUID)
	if errors.Is(err, ErrServiceUnavailable) {
		book = &domain.Book{BookUID: bookUID}
	} else if err != nil {
		return nil, err
	}
	c.books[bookUID] = *book
	return book, nil
}

func (c *catalogue) library(ctx context.Context, libraryUID uuid.UUID) (*domain.Library, error) {
	if known, ok := c.places[libraryUID]; ok {
		return &known, nil
	}

	library, err := c.libraries.GetLibrary(ctx, libraryUID)
	if errors.Is(err, ErrServiceUnavailable) {
		library = &domain.Library{LibraryUID: libraryUID}
	} else if err != nil {
		return nil, err
	}
	c.places[libraryUID] = *library
	return library, nil
}
