package library

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Impervguin/ds-lab2/gateway/internal/handler/common"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/dto"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type LibraryHandler struct {
	libraries *usecase.LibraryUseCase
	errors    common.ErrorWriter
}

func NewLibraryHandler(libraries *usecase.LibraryUseCase) *LibraryHandler {
	return &LibraryHandler{
		libraries: libraries,
		errors:    common.NewErrorWriter("handler.library"),
	}
}

func (h *LibraryHandler) Register(r chi.Router) {
	r.Route("/api/v1/libraries", func(r chi.Router) {
		r.Get("/", h.listLibraries)
		r.Get("/{libraryUid}/books", h.listBooks)
	})
}

func (h *LibraryHandler) listLibraries(w http.ResponseWriter, r *http.Request) {
	city, err := common.RequiredQuery(r, "city")
	if err != nil {
		common.BadParam(w, "city", err)
		return
	}

	page, size, ok := paging(w, r)
	if !ok {
		return
	}

	libraries, err := h.libraries.ListLibraries(r.Context(), city, page, size)
	if err != nil {
		h.errors.Write(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewLibraryPaginationResponse(libraries))
}

func (h *LibraryHandler) listBooks(w http.ResponseWriter, r *http.Request) {
	libraryUID, ok := common.PathUUID(w, r, "libraryUid")
	if !ok {
		return
	}

	page, size, ok := paging(w, r)
	if !ok {
		return
	}

	showAll, err := common.BoolQuery(r, "showAll", false)
	if err != nil {
		common.BadParam(w, "showAll", err)
		return
	}

	books, err := h.libraries.ListBooks(r.Context(), libraryUID, page, size, showAll)
	if err != nil {
		h.errors.Write(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewLibraryBookPaginationResponse(books))
}

func paging(w http.ResponseWriter, r *http.Request) (page, size int, ok bool) {
	page, err := common.IntQuery(r, "page", usecase.DefaultPage)
	if err != nil {
		common.BadParam(w, "page", err)
		return 0, 0, false
	}

	size, err = common.IntQuery(r, "size", usecase.DefaultPageSize)
	if err != nil {
		common.BadParam(w, "size", err)
		return 0, 0, false
	}
	return page, size, true
}
