package usecase

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/library/internal/domain"
	"github.com/Impervguin/ds-lab2/library/internal/logger"
)

type LibraryUseCase struct {
	libraries    domain.LibraryRepository
	books        domain.BookRepository
	libraryBooks domain.LibraryBookRepository
	log          *slog.Logger
}

func NewLibraryUseCase(
	libraries domain.LibraryRepository,
	books domain.BookRepository,
	libraryBooks domain.LibraryBookRepository,
) *LibraryUseCase {
	return &LibraryUseCase{
		libraries:    libraries,
		books:        books,
		libraryBooks: libraryBooks,
		log:          logger.Named("usecase.library"),
	}
}

func (uc *LibraryUseCase) ListLibraries(ctx context.Context, city string, page, size int) (Page[domain.Library], error) {
	page, size, limit, offset := normalizePaging(page, size)

	libraries, total, err := uc.libraries.ListByCity(ctx, city, limit, offset)
	if err != nil {
		return Page[domain.Library]{}, err
	}

	return Page[domain.Library]{Page: page, PageSize: size, TotalElements: total, Items: libraries}, nil
}

func (uc *LibraryUseCase) GetLibrary(ctx context.Context, libraryUID uuid.UUID) (*domain.Library, error) {
	return uc.libraries.Get(ctx, libraryUID)
}

func (uc *LibraryUseCase) ListBooks(
	ctx context.Context,
	libraryUID uuid.UUID,
	page, size int,
	showAll bool,
) (Page[domain.LibraryBook], error) {
	if _, err := uc.libraries.Get(ctx, libraryUID); err != nil {
		return Page[domain.LibraryBook]{}, err
	}

	page, size, limit, offset := normalizePaging(page, size)

	books, total, err := uc.libraryBooks.ListByLibrary(ctx, libraryUID, limit, offset, showAll)
	if err != nil {
		return Page[domain.LibraryBook]{}, err
	}

	return Page[domain.LibraryBook]{Page: page, PageSize: size, TotalElements: total, Items: books}, nil
}

func (uc *LibraryUseCase) GetBook(ctx context.Context, bookUID uuid.UUID) (*domain.Book, error) {
	return uc.books.Get(ctx, bookUID)
}

func (uc *LibraryUseCase) TakeBook(ctx context.Context, libraryUID, bookUID uuid.UUID) (*domain.LibraryBook, error) {
	var issued domain.LibraryBook

	_, err := uc.libraryBooks.Update(ctx, libraryUID, bookUID,
		func(_ context.Context, held []domain.LibraryBook) ([]domain.LibraryBook, error) {
			at := bestAvailable(held)
			if at < 0 {
				return nil, domain.ErrNoAvailableCopies
			}

			held[at].AvailableCount--
			issued = held[at]
			return held, nil
		})
	if err != nil {
		return nil, err
	}

	uc.log.InfoContext(ctx, "copy issued",
		slog.String("libraryUid", libraryUID.String()),
		slog.String("bookUid", bookUID.String()),
		slog.String("condition", string(issued.Condition)),
		slog.Int("availableCount", issued.AvailableCount),
	)
	return &issued, nil
}

func (uc *LibraryUseCase) ReturnBook(
	ctx context.Context,
	libraryUID, bookUID uuid.UUID,
	condition domain.BookCondition,
) (*domain.LibraryBook, error) {
	if !condition.Valid() {
		return nil, domain.ErrInvalidCondition
	}

	var shelved domain.LibraryBook

	_, err := uc.libraryBooks.Update(ctx, libraryUID, bookUID,
		func(_ context.Context, held []domain.LibraryBook) ([]domain.LibraryBook, error) {
			at := indexOf(held, condition)
			if at < 0 {
				held = append(held, domain.LibraryBook{
					Book:      held[0].Book,
					LibraryID: held[0].LibraryID,
					Condition: condition,
				})
				at = len(held) - 1
			}

			held[at].AvailableCount++
			shelved = held[at]
			return held, nil
		})
	if err != nil {
		return nil, err
	}

	uc.log.InfoContext(ctx, "copy accepted",
		slog.String("libraryUid", libraryUID.String()),
		slog.String("bookUid", bookUID.String()),
		slog.String("condition", string(shelved.Condition)),
		slog.Int("availableCount", shelved.AvailableCount),
	)
	return &shelved, nil
}

func bestAvailable(held []domain.LibraryBook) int {
	best := -1
	for i, row := range held {
		if row.AvailableCount > 0 && (best < 0 || row.Condition.Rank() < held[best].Condition.Rank()) {
			best = i
		}
	}
	return best
}

func indexOf(held []domain.LibraryBook, condition domain.BookCondition) int {
	for i, row := range held {
		if row.Condition == condition {
			return i
		}
	}
	return -1
}
