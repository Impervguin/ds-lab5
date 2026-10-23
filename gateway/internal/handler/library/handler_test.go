package library_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/library"
	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase/mocks"
)

func TestMain(m *testing.M) {
	_ = logger.Init(logger.Settings{Level: "error", Format: logger.FormatJSON, Service: "test", Output: io.Discard})
	os.Exit(m.Run())
}

const (
	libraryUIDValue = "83575e12-7ce0-48ee-9931-51919ff3c9ee"
	bookUIDValue    = "f7cdc58f-2caf-4b15-9727-f89dcc629b27"
)

var (
	libraryUID = uuid.MustParse(libraryUIDValue)
	bookUID    = uuid.MustParse(bookUIDValue)
	errService = errors.New("service is unavailable")
)

type suite struct {
	libraries *mocks.LibraryService
	routes    http.Handler
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{libraries: &mocks.LibraryService{}}

	router := chi.NewRouter()
	library.NewLibraryHandler(usecase.NewLibraryUseCase(s.libraries)).Register(router)
	s.routes = router

	t.Cleanup(func() { s.libraries.AssertExpectations(t) })
	return s
}

func (s *suite) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	s.routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func theLibraryPage() usecase.Page[domain.Library] {
	return usecase.Page[domain.Library]{
		Page:          1,
		PageSize:      10,
		TotalElements: 1,
		Items: []domain.Library{{
			LibraryUID: libraryUID,
			Name:       "Библиотека имени 7 Непьющих",
			City:       "Москва",
			Address:    "2-я Бауманская ул., д.5, стр.1",
		}},
	}
}

func theBookPage() usecase.Page[domain.LibraryBook] {
	return usecase.Page[domain.LibraryBook]{
		Page:          1,
		PageSize:      10,
		TotalElements: 1,
		Items: []domain.LibraryBook{{
			Book: domain.Book{
				BookUID: bookUID,
				Name:    "Краткий курс C++ в 7 томах",
				Author:  "Бьерн Страуструп",
				Genre:   "Научная фантастика",
			},
			Condition:      domain.ConditionExcellent,
			AvailableCount: 1,
		}},
	}
}

func TestListLibrariesAnswersWithAPageOfLibraries(t *testing.T) {
	s := newSuite(t)

	s.libraries.On("ListLibraries", mock.Anything, "Москва", 1, 10).Return(theLibraryPage(), nil)

	recorder := s.get(t, "/api/v1/libraries?city=Москва&page=1&size=10")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")

	response := decode(t, recorder)
	assert.Equal(t, float64(1), response["page"])
	assert.Equal(t, float64(10), response["pageSize"])
	assert.Equal(t, float64(1), response["totalElements"])

	items := response["items"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, libraryUIDValue, first["libraryUid"])
	assert.Equal(t, "Библиотека имени 7 Непьющих", first["name"])
	assert.Equal(t, "Москва", first["city"])
	assert.Equal(t, "2-я Бауманская ул., д.5, стр.1", first["address"])
}

func TestListLibrariesFallsBackToTheDefaultPaging(t *testing.T) {
	s := newSuite(t)

	s.libraries.On("ListLibraries", mock.Anything, "Москва", usecase.DefaultPage, usecase.DefaultPageSize).
		Return(theLibraryPage(), nil)

	recorder := s.get(t, "/api/v1/libraries?city=Москва")

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestListLibrariesDemandsTheCity(t *testing.T) {
	s := newSuite(t)

	recorder := s.get(t, "/api/v1/libraries?page=1&size=10")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "city")
}

func TestListLibrariesRejectsANonNumericPage(t *testing.T) {
	s := newSuite(t)

	recorder := s.get(t, "/api/v1/libraries?city=Москва&page=first")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "page")
}

func TestListLibrariesAnswersWithFiveHundredWhenTheServiceIsDown(t *testing.T) {
	s := newSuite(t)

	s.libraries.On("ListLibraries", mock.Anything, "Москва", 1, 10).
		Return(usecase.Page[domain.Library]{}, errService)

	recorder := s.get(t, "/api/v1/libraries?city=Москва&page=1&size=10")

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestListBooksAnswersWithAPageOfBooks(t *testing.T) {
	s := newSuite(t)

	s.libraries.On("ListBooks", mock.Anything, libraryUID, 1, 25, true).Return(theBookPage(), nil)

	recorder := s.get(t, "/api/v1/libraries/"+libraryUIDValue+"/books?page=1&size=25&showAll=true")

	require.Equal(t, http.StatusOK, recorder.Code)

	items := decode(t, recorder)["items"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, bookUIDValue, first["bookUid"])
	assert.Equal(t, "Краткий курс C++ в 7 томах", first["name"])
	assert.Equal(t, "Бьерн Страуструп", first["author"])
	assert.Equal(t, "Научная фантастика", first["genre"])
	assert.Equal(t, "EXCELLENT", first["condition"])
	assert.Equal(t, float64(1), first["availableCount"])
}

func TestListBooksHidesEmptyRowsByDefault(t *testing.T) {
	s := newSuite(t)

	s.libraries.On("ListBooks", mock.Anything, libraryUID, 1, 10, false).Return(theBookPage(), nil)

	recorder := s.get(t, "/api/v1/libraries/"+libraryUIDValue+"/books")

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestListBooksRejectsAMalformedLibraryUID(t *testing.T) {
	s := newSuite(t)

	recorder := s.get(t, "/api/v1/libraries/not-a-uuid/books")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "libraryUid")
}

func TestListBooksRejectsANonBooleanShowAll(t *testing.T) {
	s := newSuite(t)

	recorder := s.get(t, "/api/v1/libraries/"+libraryUIDValue+"/books?showAll=maybe")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "showAll")
}

func TestListBooksAnswersWithNotFoundForAnUnknownLibrary(t *testing.T) {
	s := newSuite(t)

	s.libraries.On("ListBooks", mock.Anything, libraryUID, 1, 10, false).
		Return(usecase.Page[domain.LibraryBook]{}, domain.ErrLibraryNotFound)

	recorder := s.get(t, "/api/v1/libraries/"+libraryUIDValue+"/books")

	require.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Library not found")
}
