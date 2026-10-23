package usecase_test

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/library/internal/domain"
	"github.com/Impervguin/ds-lab2/library/internal/domain/mocks"
	"github.com/Impervguin/ds-lab2/library/internal/logger"
	"github.com/Impervguin/ds-lab2/library/internal/usecase"
)

func TestMain(m *testing.M) {
	_ = logger.Init(logger.Settings{Level: "error", Format: logger.FormatJSON, Service: "test", Output: io.Discard})
	os.Exit(m.Run())
}

type suite struct {
	libraries    *mocks.LibraryRepository
	books        *mocks.BookRepository
	libraryBooks *mocks.LibraryBookRepository
	useCase      *usecase.LibraryUseCase
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{
		libraries:    &mocks.LibraryRepository{},
		books:        &mocks.BookRepository{},
		libraryBooks: &mocks.LibraryBookRepository{},
	}
	s.useCase = usecase.NewLibraryUseCase(s.libraries, s.books, s.libraryBooks)

	t.Cleanup(func() {
		s.libraries.AssertExpectations(t)
		s.books.AssertExpectations(t)
		s.libraryBooks.AssertExpectations(t)
	})
	return s
}

func (s *suite) holding(t *testing.T, libraryUID, bookUID uuid.UUID, held []domain.LibraryBook) *changes {
	t.Helper()

	result := &changes{}
	s.libraryBooks.On("Update", mock.Anything, libraryUID, bookUID, mock.Anything).
		Run(func(args mock.Arguments) {
			updFunc, ok := args.Get(3).(domain.LibraryBooksUpdateFunc)
			require.True(t, ok, "Update was called without a callback")
			result.rows, result.err = updFunc(context.Background(), held)
		}).
		Return(nil, nil)
	return result
}

type changes struct {
	rows []domain.LibraryBook
	err  error
}

func (c *changes) count(condition domain.BookCondition) int {
	for _, row := range c.rows {
		if row.Condition == condition {
			return row.AvailableCount
		}
	}
	return -1
}

func copies(condition domain.BookCondition, available int) domain.LibraryBook {
	return domain.LibraryBook{
		Book:           domain.Book{ID: 7, Name: "Краткий курс C++ в 7 томах"},
		LibraryID:      1,
		Condition:      condition,
		AvailableCount: available,
	}
}

func TestListLibrariesReportsTheWholeCollection(t *testing.T) {
	s := newSuite(t)
	found := []domain.Library{{Name: "Библиотека имени 7 Непьющих", City: "Москва"}}
	s.libraries.On("ListByCity", mock.Anything, "Москва", 10, 10).Return(found, 42, nil)

	page, err := s.useCase.ListLibraries(context.Background(), "Москва", 2, 10)

	require.NoError(t, err)
	assert.Equal(t, 2, page.Page)
	assert.Equal(t, 10, page.PageSize)
	assert.Equal(t, 42, page.TotalElements)
	assert.Equal(t, found, page.Items)
}

func TestListLibrariesNormalizesPaging(t *testing.T) {
	cases := []struct {
		name                  string
		page, size            int
		wantPage, wantSize    int
		wantLimit, wantOffset int
	}{
		{"defaults are kept", 1, 10, 1, 10, 10, 0},
		{"second page is offset by one page", 3, 25, 3, 25, 25, 50},
		{"page zero is the first page", 0, 10, 1, 10, 10, 0},
		{"negative page is the first page", -5, 10, 1, 10, 10, 0},
		{"missing size falls back", 1, 0, 1, usecase.DefaultPageSize, usecase.DefaultPageSize, 0},
		{"oversized page is clamped", 1, 1000, 1, usecase.MaxPageSize, usecase.MaxPageSize, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSuite(t)
			s.libraries.On("ListByCity", mock.Anything, "Москва", tc.wantLimit, tc.wantOffset).
				Return([]domain.Library{}, 0, nil)

			page, err := s.useCase.ListLibraries(context.Background(), "Москва", tc.page, tc.size)

			require.NoError(t, err)
			assert.Equal(t, tc.wantPage, page.Page)
			assert.Equal(t, tc.wantSize, page.PageSize)
		})
	}
}

func TestListLibrariesPropagatesRepositoryFailure(t *testing.T) {
	s := newSuite(t)
	failure := errors.New("connection refused")
	s.libraries.On("ListByCity", mock.Anything, "Москва", 10, 0).Return(nil, 0, failure)

	_, err := s.useCase.ListLibraries(context.Background(), "Москва", 1, 10)

	require.ErrorIs(t, err, failure)
}

func TestListBooksPassesShowAllThrough(t *testing.T) {
	for _, showAll := range []bool{false, true} {
		s := newSuite(t)
		libraryUID := uuid.New()
		s.libraries.On("Get", mock.Anything, libraryUID).Return(&domain.Library{ID: 1}, nil)
		s.libraryBooks.On("ListByLibrary", mock.Anything, libraryUID, 10, 0, showAll).
			Return([]domain.LibraryBook{}, 0, nil)

		_, err := s.useCase.ListBooks(context.Background(), libraryUID, 1, 10, showAll)

		require.NoError(t, err)
	}
}

func TestListBooksOfUnknownLibraryIsNotAnEmptyPage(t *testing.T) {
	s := newSuite(t)
	libraryUID := uuid.New()
	s.libraries.On("Get", mock.Anything, libraryUID).Return(nil, domain.ErrLibraryNotFound)

	_, err := s.useCase.ListBooks(context.Background(), libraryUID, 1, 10, true)

	require.ErrorIs(t, err, domain.ErrLibraryNotFound)
	s.libraryBooks.AssertNotCalled(t, "ListByLibrary", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTakeBookIssuesTheBestConditionAvailable(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	changed := s.holding(t, libraryUID, bookUID, []domain.LibraryBook{
		copies(domain.ConditionBad, 5),
		copies(domain.ConditionExcellent, 1),
		copies(domain.ConditionGood, 5),
	})

	issued, err := s.useCase.TakeBook(context.Background(), libraryUID, bookUID)

	require.NoError(t, err)
	assert.Equal(t, domain.ConditionExcellent, issued.Condition)
	assert.Equal(t, 0, issued.AvailableCount)

	require.NoError(t, changed.err)
	assert.Equal(t, 0, changed.count(domain.ConditionExcellent))
	assert.Equal(t, 5, changed.count(domain.ConditionGood))
	assert.Equal(t, 5, changed.count(domain.ConditionBad))
}

func TestTakeBookFallsBackToAWorseCondition(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	changed := s.holding(t, libraryUID, bookUID, []domain.LibraryBook{
		copies(domain.ConditionExcellent, 0),
		copies(domain.ConditionGood, 1),
		copies(domain.ConditionBad, 3),
	})

	issued, err := s.useCase.TakeBook(context.Background(), libraryUID, bookUID)

	require.NoError(t, err)
	assert.Equal(t, domain.ConditionGood, issued.Condition)
	assert.Equal(t, 0, changed.count(domain.ConditionGood))
	assert.Equal(t, 3, changed.count(domain.ConditionBad))
}

func TestTakeBookWithoutFreeCopies(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	held := []domain.LibraryBook{copies(domain.ConditionExcellent, 0), copies(domain.ConditionBad, 0)}
	s.libraryBooks.On("Update", mock.Anything, libraryUID, bookUID, mock.Anything).
		Run(func(args mock.Arguments) {
			updFunc := args.Get(3).(domain.LibraryBooksUpdateFunc)
			rows, err := updFunc(context.Background(), held)
			require.ErrorIs(t, err, domain.ErrNoAvailableCopies)
			assert.Nil(t, rows)
		}).
		Return(nil, domain.ErrNoAvailableCopies)

	_, err := s.useCase.TakeBook(context.Background(), libraryUID, bookUID)

	require.ErrorIs(t, err, domain.ErrNoAvailableCopies)
	assert.Equal(t, 0, held[0].AvailableCount)
	assert.Equal(t, 0, held[1].AvailableCount)
}

func TestTakeBookOfABookTheLibraryDoesNotHold(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	s.libraryBooks.On("Update", mock.Anything, libraryUID, bookUID, mock.Anything).
		Return(nil, domain.ErrBookNotFound)

	_, err := s.useCase.TakeBook(context.Background(), libraryUID, bookUID)

	require.ErrorIs(t, err, domain.ErrBookNotFound)
}

func TestTakeBookFromUnknownLibrary(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	s.libraryBooks.On("Update", mock.Anything, libraryUID, bookUID, mock.Anything).
		Return(nil, domain.ErrLibraryNotFound)

	_, err := s.useCase.TakeBook(context.Background(), libraryUID, bookUID)

	require.ErrorIs(t, err, domain.ErrLibraryNotFound)
}

func TestReturnBookJoinsTheRowOfItsCondition(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	changed := s.holding(t, libraryUID, bookUID, []domain.LibraryBook{
		copies(domain.ConditionExcellent, 0),
		copies(domain.ConditionBad, 1),
	})

	shelved, err := s.useCase.ReturnBook(context.Background(), libraryUID, bookUID, domain.ConditionBad)

	require.NoError(t, err)
	assert.Equal(t, domain.ConditionBad, shelved.Condition)
	assert.Equal(t, 2, shelved.AvailableCount)

	require.Len(t, changed.rows, 2)
	assert.Equal(t, 2, changed.count(domain.ConditionBad))
	assert.Equal(t, 0, changed.count(domain.ConditionExcellent))
}

func TestReturnBookOpensARowForAConditionTheLibraryNeverHeld(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	changed := s.holding(t, libraryUID, bookUID, []domain.LibraryBook{copies(domain.ConditionExcellent, 0)})

	shelved, err := s.useCase.ReturnBook(context.Background(), libraryUID, bookUID, domain.ConditionBad)

	require.NoError(t, err)
	assert.Equal(t, domain.ConditionBad, shelved.Condition)
	assert.Equal(t, 1, shelved.AvailableCount)

	require.Len(t, changed.rows, 2)
	assert.Equal(t, 1, changed.count(domain.ConditionBad))

	opened := changed.rows[1]
	assert.Equal(t, int64(1), opened.LibraryID)
	assert.Equal(t, int64(7), opened.Book.ID)
}

func TestReturnBookRejectsAnUnknownCondition(t *testing.T) {
	s := newSuite(t)

	_, err := s.useCase.ReturnBook(context.Background(), uuid.New(), uuid.New(), domain.BookCondition("DESTROYED"))

	require.ErrorIs(t, err, domain.ErrInvalidCondition)
	s.libraryBooks.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReturnBookOfABookTheLibraryDoesNotHold(t *testing.T) {
	s := newSuite(t)
	libraryUID, bookUID := uuid.New(), uuid.New()
	s.libraryBooks.On("Update", mock.Anything, libraryUID, bookUID, mock.Anything).
		Return(nil, domain.ErrBookNotFound)

	_, err := s.useCase.ReturnBook(context.Background(), libraryUID, bookUID, domain.ConditionGood)

	require.ErrorIs(t, err, domain.ErrBookNotFound)
}

func TestGetLibraryPropagatesNotFound(t *testing.T) {
	s := newSuite(t)
	libraryUID := uuid.New()
	s.libraries.On("Get", mock.Anything, libraryUID).Return(nil, domain.ErrLibraryNotFound)

	_, err := s.useCase.GetLibrary(context.Background(), libraryUID)

	require.ErrorIs(t, err, domain.ErrLibraryNotFound)
}

func TestGetBookPropagatesNotFound(t *testing.T) {
	s := newSuite(t)
	bookUID := uuid.New()
	s.books.On("Get", mock.Anything, bookUID).Return(nil, domain.ErrBookNotFound)

	_, err := s.useCase.GetBook(context.Background(), bookUID)

	require.ErrorIs(t, err, domain.ErrBookNotFound)
}
