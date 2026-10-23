package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

func TestListLibrariesReturnsWhatTheLibraryServiceReports(t *testing.T) {
	s := newSuite(t)
	found := usecase.Page[domain.Library]{Page: 1, PageSize: 10, TotalElements: 1, Items: []domain.Library{theLibrary()}}

	s.libraries.On("ListLibraries", mock.Anything, "Москва", 1, 10).Return(found, nil)

	libraries, err := usecase.NewLibraryUseCase(s.libraries).ListLibraries(context.Background(), "Москва", 1, 10)

	require.NoError(t, err)
	assert.Equal(t, found, libraries)
}

func TestListBooksPassesTheShowAllFlagOn(t *testing.T) {
	s := newSuite(t)
	found := usecase.Page[domain.LibraryBook]{
		Page:          1,
		PageSize:      10,
		TotalElements: 1,
		Items: []domain.LibraryBook{
			{Book: theBook(), Condition: domain.ConditionExcellent, AvailableCount: 1},
		},
	}

	s.libraries.On("ListBooks", mock.Anything, libraryUID, 1, 10, true).Return(found, nil)

	books, err := usecase.NewLibraryUseCase(s.libraries).ListBooks(context.Background(), libraryUID, 1, 10, true)

	require.NoError(t, err)
	assert.Equal(t, found, books)
}

func TestListBooksReportsAnUnknownLibrary(t *testing.T) {
	s := newSuite(t)

	s.libraries.On("ListBooks", mock.Anything, libraryUID, 1, 10, false).
		Return(usecase.Page[domain.LibraryBook]{}, domain.ErrLibraryNotFound)

	_, err := usecase.NewLibraryUseCase(s.libraries).ListBooks(context.Background(), libraryUID, 1, 10, false)

	assert.ErrorIs(t, err, domain.ErrLibraryNotFound)
}
