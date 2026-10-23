package library

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/library/internal/domain"
	"github.com/Impervguin/ds-lab2/library/internal/handler/common"
	"github.com/Impervguin/ds-lab2/library/internal/handler/library/dto"
	"github.com/Impervguin/ds-lab2/library/internal/logger"
	"github.com/Impervguin/ds-lab2/library/internal/usecase"
)

type LibraryHandler struct {
	libraries *usecase.LibraryUseCase
	validate  *validator.Validate
	log       *slog.Logger
}

func NewLibraryHandler(libraries *usecase.LibraryUseCase) *LibraryHandler {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &LibraryHandler{
		libraries: libraries,
		validate:  validate,
		log:       logger.Named("handler.library"),
	}
}

func (h *LibraryHandler) Register(r chi.Router) {
	r.Route("/api/v1/libraries", func(r chi.Router) {
		r.Get("/", h.listLibraries)

		r.Route("/{libraryUid}", func(r chi.Router) {
			r.Get("/", h.getLibrary)
			r.Get("/books", h.listBooks)
			r.Post("/books/{bookUid}/take", h.takeBook)
			r.Post("/books/{bookUid}/return", h.returnBook)
		})
	})

	r.Get("/api/v1/books/{bookUid}", h.getBook)
}

func (h *LibraryHandler) listLibraries(w http.ResponseWriter, r *http.Request) {
	city, err := common.RequiredQuery(r, "city")
	if err != nil {
		badQueryParam(w, "city", err)
		return
	}

	page, err := common.IntQuery(r, "page", usecase.DefaultPage)
	if err != nil {
		badQueryParam(w, "page", err)
		return
	}

	size, err := common.IntQuery(r, "size", usecase.DefaultPageSize)
	if err != nil {
		badQueryParam(w, "size", err)
		return
	}

	libraries, err := h.libraries.ListLibraries(r.Context(), city, page, size)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewLibraryPaginationResponse(libraries))
}

func (h *LibraryHandler) getLibrary(w http.ResponseWriter, r *http.Request) {
	libraryUID, err := h.pathUUID(w, r, "libraryUid")
	if err != nil {
		return
	}

	library, err := h.libraries.GetLibrary(r.Context(), libraryUID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewLibraryResponse(*library))
}

func (h *LibraryHandler) listBooks(w http.ResponseWriter, r *http.Request) {
	libraryUID, err := h.pathUUID(w, r, "libraryUid")
	if err != nil {
		return
	}

	page, err := common.IntQuery(r, "page", usecase.DefaultPage)
	if err != nil {
		badQueryParam(w, "page", err)
		return
	}

	size, err := common.IntQuery(r, "size", usecase.DefaultPageSize)
	if err != nil {
		badQueryParam(w, "size", err)
		return
	}

	showAll, err := common.BoolQuery(r, "showAll", false)
	if err != nil {
		badQueryParam(w, "showAll", err)
		return
	}

	books, err := h.libraries.ListBooks(r.Context(), libraryUID, page, size, showAll)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewLibraryBookPaginationResponse(books))
}

func (h *LibraryHandler) getBook(w http.ResponseWriter, r *http.Request) {
	bookUID, err := h.pathUUID(w, r, "bookUid")
	if err != nil {
		return
	}

	book, err := h.libraries.GetBook(r.Context(), bookUID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewBookResponse(*book))
}

func (h *LibraryHandler) takeBook(w http.ResponseWriter, r *http.Request) {
	libraryUID, err := h.pathUUID(w, r, "libraryUid")
	if err != nil {
		return
	}

	bookUID, err := h.pathUUID(w, r, "bookUid")
	if err != nil {
		return
	}

	taken, err := h.libraries.TakeBook(r.Context(), libraryUID, bookUID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewTakeBookResponse(*taken))
}

func (h *LibraryHandler) returnBook(w http.ResponseWriter, r *http.Request) {
	libraryUID, err := h.pathUUID(w, r, "libraryUid")
	if err != nil {
		return
	}

	bookUID, err := h.pathUUID(w, r, "bookUid")
	if err != nil {
		return
	}

	var request dto.ReturnBookRequest
	if err := common.DecodeJSON(r, &request); err != nil {
		common.WriteValidationError(w, err.Error(), nil)
		return
	}
	if err := h.validate.Struct(&request); err != nil {
		common.WriteValidationError(w, "request validation failed", validationErrors(err))
		return
	}

	returned, err := h.libraries.ReturnBook(r.Context(), libraryUID, bookUID, domain.BookCondition(request.Condition))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, dto.NewReturnBookResponse(*returned))
}

func (h *LibraryHandler) pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, error) {
	value, err := common.UUIDParam(r, name)
	if err != nil {
		common.WriteValidationError(w, "request validation failed", []common.ErrorDescription{
			{Field: name, Error: err.Error()},
		})
		return uuid.Nil, err
	}
	return value, nil
}

func (h *LibraryHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrLibraryNotFound):
		common.WriteError(w, http.StatusNotFound, "Library not found")
	case errors.Is(err, domain.ErrBookNotFound):
		common.WriteError(w, http.StatusNotFound, "Book not found")
	case errors.Is(err, domain.ErrNoAvailableCopies):
		common.WriteError(w, http.StatusConflict, "No available copies of the book left")
	case errors.Is(err, domain.ErrInvalidCondition):
		common.WriteValidationError(w, "request validation failed", []common.ErrorDescription{
			{Field: "condition", Error: "unknown book condition"},
		})
	default:
		h.log.ErrorContext(r.Context(), "request failed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
		)
		common.WriteError(w, http.StatusInternalServerError, "Internal server error")
	}
}

func badQueryParam(w http.ResponseWriter, name string, err error) {
	common.WriteValidationError(w, "request validation failed", []common.ErrorDescription{
		{Field: name, Error: err.Error()},
	})
}

func validationErrors(err error) []common.ErrorDescription {
	var invalid validator.ValidationErrors
	if !errors.As(err, &invalid) {
		return []common.ErrorDescription{{Error: err.Error()}}
	}

	described := make([]common.ErrorDescription, 0, len(invalid))
	for _, fieldError := range invalid {
		described = append(described, common.ErrorDescription{
			Field: fieldError.Field(),
			Error: "failed on the " + fieldError.Tag() + " rule",
		})
	}
	return described
}
