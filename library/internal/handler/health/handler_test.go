package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/Impervguin/ds-lab2/library/internal/handler/health"
)

func TestHealthIsAlwaysOK(t *testing.T) {
	router := chi.NewRouter()
	health.NewHealthHandler().Register(router)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/manage/health", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}
