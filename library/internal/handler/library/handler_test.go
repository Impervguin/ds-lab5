package library_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/library/internal/domain"
	"github.com/Impervguin/ds-lab2/library/internal/domain/mocks"
	"github.com/Impervguin/ds-lab2/library/internal/handler/library"
	"github.com/Impervguin/ds-lab2/library/internal/logger"
	"github.com/Impervguin/ds-lab2/library/internal/usecase"
)

func TestMain(m *testing.M) {
	_ = logger.Init(logger.Settings{Level: "error", Format: logger.FormatJSON, Service: "test", Output: io.Discard})
	os.Exit(m.Run())
}

const (
	libraryUID = "83575e12-7ce0-48ee-9931-51919ff3c9ee"
	bookUID    = "f7cdc58f-2caf-4b15-9727-f89dcc629b27"
)

type suite struct {
	libraries    *mocks.LibraryRepository
	books        *mocks.BookRepository
	libraryBooks *mocks.LibraryBookRepository
	routes       http.Handler
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{
		libraries:    &mocks.LibraryRepository{},
		books:        &mocks.BookRepository{},
		libraryBooks: &mocks.LibraryBookRepository{},
	}

	router := chi.NewRouter()
	library.NewLibraryHandler(usecase.NewLibraryUseCase(s.libraries, s.books, s.libraryBooks)).Register(router)
	s.routes = router

	t.Cleanup(func() {
		s.libraries.AssertExpectations(t)
		s.books.AssertExpectations(t)
		s.libraryBooks.AssertExpectations(t)
	})
	return s
}

func (s *suite) do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(method, target, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	s.routes.ServeHTTP(recorder, request)
	return recorder
}

func (s *suite) libraryExists(t *testing.T) uuid.UUID {
	t.Helper()

	parsed := uuid.MustParse(libraryUID)
	s.libraries.On("Get", mock.Anything, parsed).Return(&domain.Library{
		ID:         1,
		LibraryUID: parsed,
		Name:       "Библиотека имени 7 Непьющих",
		City:       "Москва",
		Address:    "2-я Бауманская ул., д.5, стр.1",
	}, nil)
	return parsed
}

func (s *suite) holding(t *testing.T, held []domain.LibraryBook) {
	t.Helper()

	s.libraryBooks.On("Update", mock.Anything, uuid.MustParse(libraryUID), uuid.MustParse(bookUID), mock.Anything).
		Run(func(args mock.Arguments) {
			updFunc, ok := args.Get(3).(domain.LibraryBooksUpdateFunc)
			require.True(t, ok, "Update was called without a callback")
			_, _ = updFunc(context.Background(), held)
		}).
		Return(nil, nil)
}

func (s *suite) holdingFails(err error) {
	s.libraryBooks.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, err)
}

func theBook() domain.Book {
	return domain.Book{
		ID:      7,
		BookUID: uuid.MustParse(bookUID),
		Name:    "Краткий курс C++ в 7 томах",
		Author:  "Бьерн Страуструп",
		Genre:   "Научная фантастика",
	}
}

func copies(condition domain.BookCondition, available int) domain.LibraryBook {
	return domain.LibraryBook{Book: theBook(), LibraryID: 1, Condition: condition, AvailableCount: available}
}

func TestListLibraries(t *testing.T) {
	s := newSuite(t)
	s.libraries.On("ListByCity", mock.Anything, "Москва", 10, 0).Return([]domain.Library{{
		LibraryUID: uuid.MustParse(libraryUID),
		Name:       "Библиотека имени 7 Непьющих",
		City:       "Москва",
		Address:    "2-я Бауманская ул., д.5, стр.1",
	}}, 1, nil)

	response := s.do(t, http.MethodGet, "/api/v1/libraries?city=Москва&page=1&size=10", "")

	require.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Header().Get("Content-Type"), "application/json")
	assert.JSONEq(t, `{
		"page": 1,
		"pageSize": 10,
		"totalElements": 1,
		"items": [{
			"libraryUid": "83575e12-7ce0-48ee-9931-51919ff3c9ee",
			"name": "Библиотека имени 7 Непьющих",
			"address": "2-я Бауманская ул., д.5, стр.1",
			"city": "Москва"
		}]
	}`, response.Body.String())
}

func TestListLibrariesWithoutCity(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodGet, "/api/v1/libraries", "")

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"field":"city"`)
	s.libraries.AssertNotCalled(t, "ListByCity", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestListLibrariesWithAPageThatIsNotANumber(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodGet, "/api/v1/libraries?city=Москва&page=abc", "")

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"field":"page"`)
}

func TestListLibrariesWhenTheRepositoryFails(t *testing.T) {
	s := newSuite(t)
	s.libraries.On("ListByCity", mock.Anything, "Москва", 10, 0).
		Return(nil, 0, errors.New("connection refused"))

	response := s.do(t, http.MethodGet, "/api/v1/libraries?city=Москва", "")

	require.Equal(t, http.StatusInternalServerError, response.Code)
	assert.JSONEq(t, `{"message": "Internal server error"}`, response.Body.String())
}

func TestGetLibrary(t *testing.T) {
	s := newSuite(t)
	s.libraryExists(t)

	response := s.do(t, http.MethodGet, "/api/v1/libraries/"+libraryUID, "")

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{
		"libraryUid": "83575e12-7ce0-48ee-9931-51919ff3c9ee",
		"name": "Библиотека имени 7 Непьющих",
		"address": "2-я Бауманская ул., д.5, стр.1",
		"city": "Москва"
	}`, response.Body.String())
}

func TestGetUnknownLibrary(t *testing.T) {
	s := newSuite(t)
	s.libraries.On("Get", mock.Anything, mock.Anything).Return(nil, domain.ErrLibraryNotFound)

	response := s.do(t, http.MethodGet, "/api/v1/libraries/"+libraryUID, "")

	require.Equal(t, http.StatusNotFound, response.Code)
	assert.JSONEq(t, `{"message": "Library not found"}`, response.Body.String())
}

func TestGetLibraryWithAMalformedUID(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodGet, "/api/v1/libraries/not-a-uuid", "")

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"field":"libraryUid"`)
	s.libraries.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestListBooksShowsEveryConditionSeparately(t *testing.T) {
	s := newSuite(t)
	parsed := s.libraryExists(t)
	s.libraryBooks.On("ListByLibrary", mock.Anything, parsed, 25, 0, true).
		Return([]domain.LibraryBook{copies(domain.ConditionExcellent, 0), copies(domain.ConditionBad, 1)}, 2, nil)

	response := s.do(t, http.MethodGet, "/api/v1/libraries/"+libraryUID+"/books?page=1&size=25&showAll=true", "")

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{
		"page": 1,
		"pageSize": 25,
		"totalElements": 2,
		"items": [
			{
				"bookUid": "f7cdc58f-2caf-4b15-9727-f89dcc629b27",
				"name": "Краткий курс C++ в 7 томах",
				"author": "Бьерн Страуструп",
				"genre": "Научная фантастика",
				"condition": "EXCELLENT",
				"availableCount": 0
			},
			{
				"bookUid": "f7cdc58f-2caf-4b15-9727-f89dcc629b27",
				"name": "Краткий курс C++ в 7 томах",
				"author": "Бьерн Страуструп",
				"genre": "Научная фантастика",
				"condition": "BAD",
				"availableCount": 1
			}
		]
	}`, response.Body.String())
}

func TestListBooksDefaultsToHidingEmptyRows(t *testing.T) {
	s := newSuite(t)
	parsed := s.libraryExists(t)
	s.libraryBooks.On("ListByLibrary", mock.Anything, parsed, usecase.DefaultPageSize, 0, false).
		Return([]domain.LibraryBook{}, 0, nil)

	response := s.do(t, http.MethodGet, "/api/v1/libraries/"+libraryUID+"/books", "")

	require.Equal(t, http.StatusOK, response.Code)
}

func TestListBooksWithAShowAllThatIsNotABoolean(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodGet, "/api/v1/libraries/"+libraryUID+"/books?showAll=maybe", "")

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"field":"showAll"`)
	s.libraries.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestGetBook(t *testing.T) {
	s := newSuite(t)
	book := theBook()
	s.books.On("Get", mock.Anything, book.BookUID).Return(&book, nil)

	response := s.do(t, http.MethodGet, "/api/v1/books/"+bookUID, "")

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{
		"bookUid": "f7cdc58f-2caf-4b15-9727-f89dcc629b27",
		"name": "Краткий курс C++ в 7 томах",
		"author": "Бьерн Страуструп",
		"genre": "Научная фантастика"
	}`, response.Body.String())
}

func TestGetUnknownBook(t *testing.T) {
	s := newSuite(t)
	s.books.On("Get", mock.Anything, mock.Anything).Return(nil, domain.ErrBookNotFound)

	response := s.do(t, http.MethodGet, "/api/v1/books/"+bookUID, "")

	require.Equal(t, http.StatusNotFound, response.Code)
	assert.JSONEq(t, `{"message": "Book not found"}`, response.Body.String())
}

func TestTakeBook(t *testing.T) {
	s := newSuite(t)
	s.holding(t, []domain.LibraryBook{copies(domain.ConditionExcellent, 1)})

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/take", "")

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"condition": "EXCELLENT", "availableCount": 0}`, response.Body.String())
}

func TestTakeBookWithoutFreeCopies(t *testing.T) {
	s := newSuite(t)
	s.holdingFails(domain.ErrNoAvailableCopies)

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/take", "")

	require.Equal(t, http.StatusConflict, response.Code)
	assert.JSONEq(t, `{"message": "No available copies of the book left"}`, response.Body.String())
}

func TestTakeBookTheLibraryDoesNotHold(t *testing.T) {
	s := newSuite(t)
	s.holdingFails(domain.ErrBookNotFound)

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/take", "")

	require.Equal(t, http.StatusNotFound, response.Code)
	assert.JSONEq(t, `{"message": "Book not found"}`, response.Body.String())
}

func TestTakeBookFromUnknownLibrary(t *testing.T) {
	s := newSuite(t)
	s.holdingFails(domain.ErrLibraryNotFound)

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/take", "")

	require.Equal(t, http.StatusNotFound, response.Code)
	assert.JSONEq(t, `{"message": "Library not found"}`, response.Body.String())
}

func TestTakeBookWithAMalformedBookUID(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/not-a-uuid/take", "")

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"field":"bookUid"`)
}

func TestTakeBookWhenTheRepositoryFails(t *testing.T) {
	s := newSuite(t)
	s.holdingFails(errors.New("connection refused"))

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/take", "")

	require.Equal(t, http.StatusInternalServerError, response.Code)
	assert.JSONEq(t, `{"message": "Internal server error"}`, response.Body.String())
}

func TestReturnBook(t *testing.T) {
	s := newSuite(t)
	s.holding(t, []domain.LibraryBook{copies(domain.ConditionBad, 1)})

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/return", `{"condition": "BAD"}`)

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"condition": "BAD", "availableCount": 2}`, response.Body.String())
}

func TestReturnBookInAnUnknownCondition(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodPost,
		"/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/return", `{"condition": "DESTROYED"}`)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"field":"condition"`)
	s.libraries.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestReturnBookWithAnEmptyBody(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodPost, "/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/return", "")

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "request body is empty")
}

func TestReturnBookWithAnUnknownField(t *testing.T) {
	s := newSuite(t)

	response := s.do(t, http.MethodPost,
		"/api/v1/libraries/"+libraryUID+"/books/"+bookUID+"/return", `{"condition": "GOOD", "date": "2021-10-11"}`)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `unknown field \"date\"`)
}
