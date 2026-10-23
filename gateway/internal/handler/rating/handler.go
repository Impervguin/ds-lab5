package rating

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Impervguin/ds-lab2/gateway/internal/handler/common"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/dto"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type RatingHandler struct {
	ratings *usecase.RatingUseCase
	errors  common.ErrorWriter
}

func NewRatingHandler(ratings *usecase.RatingUseCase) *RatingHandler {
	return &RatingHandler{
		ratings: ratings,
		errors:  common.NewErrorWriter("handler.rating"),
	}
}

func (h *RatingHandler) Register(r chi.Router) {
	r.Get("/api/v1/rating", h.getRating)
}

func (h *RatingHandler) getRating(w http.ResponseWriter, r *http.Request) {
	username, ok := common.Username(w, r)
	if !ok {
		return
	}

	rating, err := h.ratings.GetRating(r.Context(), username)
	if err != nil {
		h.errors.Write(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewUserRatingResponse(*rating))
}
