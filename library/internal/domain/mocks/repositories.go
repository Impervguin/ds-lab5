package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Impervguin/ds-lab2/library/internal/domain"
)

type LibraryRepository struct {
	mock.Mock
}

var _ domain.LibraryRepository = (*LibraryRepository)(nil)

func (m *LibraryRepository) ListByCity(ctx context.Context, city string, limit, offset int) ([]domain.Library, int, error) {
	args := m.Called(ctx, city, limit, offset)
	libraries, _ := args.Get(0).([]domain.Library)
	return libraries, args.Int(1), args.Error(2)
}

func (m *LibraryRepository) Get(ctx context.Context, libraryUID uuid.UUID) (*domain.Library, error) {
	args := m.Called(ctx, libraryUID)
	library, _ := args.Get(0).(*domain.Library)
	return library, args.Error(1)
}

type BookRepository struct {
	mock.Mock
}

var _ domain.BookRepository = (*BookRepository)(nil)

func (m *BookRepository) Get(ctx context.Context, bookUID uuid.UUID) (*domain.Book, error) {
	args := m.Called(ctx, bookUID)
	book, _ := args.Get(0).(*domain.Book)
	return book, args.Error(1)
}

type LibraryBookRepository struct {
	mock.Mock
}

var _ domain.LibraryBookRepository = (*LibraryBookRepository)(nil)

func (m *LibraryBookRepository) ListByLibrary(
	ctx context.Context,
	libraryUID uuid.UUID,
	limit, offset int,
	includeEmpty bool,
) ([]domain.LibraryBook, int, error) {
	args := m.Called(ctx, libraryUID, limit, offset, includeEmpty)
	books, _ := args.Get(0).([]domain.LibraryBook)
	return books, args.Int(1), args.Error(2)
}

func (m *LibraryBookRepository) Update(
	ctx context.Context,
	libraryUID, bookUID uuid.UUID,
	updFunc domain.LibraryBooksUpdateFunc,
) ([]domain.LibraryBook, error) {
	args := m.Called(ctx, libraryUID, bookUID, updFunc)
	updated, _ := args.Get(0).([]domain.LibraryBook)
	return updated, args.Error(1)
}
