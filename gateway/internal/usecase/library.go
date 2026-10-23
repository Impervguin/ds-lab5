package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
)

type LibraryUseCase struct {
	libraries LibraryService
}

func NewLibraryUseCase(libraries LibraryService) *LibraryUseCase {
	return &LibraryUseCase{libraries: libraries}
}

func (uc *LibraryUseCase) ListLibraries(ctx context.Context, city string, page, size int) (Page[domain.Library], error) {
	libraries, err := uc.libraries.ListLibraries(ctx, city, page, size)
	if err != nil {
		return Page[domain.Library]{}, unavailable(err, ErrLibraryUnavailable)
	}
	return libraries, nil
}

func (uc *LibraryUseCase) ListBooks(
	ctx context.Context,
	libraryUID uuid.UUID,
	page, size int,
	showAll bool,
) (Page[domain.LibraryBook], error) {
	books, err := uc.libraries.ListBooks(ctx, libraryUID, page, size, showAll)
	if err != nil {
		return Page[domain.LibraryBook]{}, unavailable(err, ErrLibraryUnavailable)
	}
	return books, nil
}
