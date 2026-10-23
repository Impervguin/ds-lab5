package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Register(r chi.Router) {
	r.Get("/manage/health", h.health)
}

func (h *HealthHandler) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
