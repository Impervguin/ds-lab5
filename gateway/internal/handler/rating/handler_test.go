package rating_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/rating"
	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase/mocks"
)

func TestMain(m *testing.M) {
	_ = logger.Init(logger.Settings{Level: "error", Format: logger.FormatJSON, Service: "test", Output: io.Discard})
	os.Exit(m.Run())
}

const username = "Test Max"

var errService = errors.New("service is unavailable")

type suite struct {
	ratings *mocks.RatingService
	routes  http.Handler
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{ratings: &mocks.RatingService{}}

	router := chi.NewRouter()
	rating.NewRatingHandler(usecase.NewRatingUseCase(s.ratings)).Register(router)
	s.routes = router

	t.Cleanup(func() { s.ratings.AssertExpectations(t) })
	return s
}

func (s *suite) get(t *testing.T, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/rating", nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}

	recorder := httptest.NewRecorder()
	s.routes.ServeHTTP(recorder, request)
	return recorder
}

func TestGetRatingAnswersWithStars(t *testing.T) {
	s := newSuite(t)

	s.ratings.On("GetRating", mock.Anything, username).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)

	recorder := s.get(t, map[string]string{"X-User-Name": username})

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")

	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, float64(75), response["stars"])
	assert.Equal(t, float64(25), response["maxBooks"])
}

func TestGetRatingDemandsTheUserName(t *testing.T) {
	s := newSuite(t)

	recorder := s.get(t, nil)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "X-User-Name")
}

func TestGetRatingAnswersWithFiveHundredWhenTheServiceIsDown(t *testing.T) {
	s := newSuite(t)

	s.ratings.On("GetRating", mock.Anything, username).Return(nil, errService)

	recorder := s.get(t, map[string]string{"X-User-Name": username})

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Internal server error")
}
