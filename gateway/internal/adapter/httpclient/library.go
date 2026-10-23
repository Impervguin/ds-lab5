package httpclient

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type LibraryClient struct {
	transport HttpDoer
	retryer   *HttpRetryer
}

var _ usecase.LibraryService = (*LibraryClient)(nil)

func NewLibraryClient(transport HttpDoer, retryer *HttpRetryer) *LibraryClient {
	return &LibraryClient{transport: transport, retryer: retryer}
}

func (c *LibraryClient) ListLibraries(
	ctx context.Context,
	city string,
	page, size int,
) (usecase.Page[domain.Library], error) {
	query := paging(page, size)
	query.Set("city", city)

	var response pageResponse[libraryResponse]
	if err := c.transport.do(ctx, call{
		method: http.MethodGet,
		path:   "/api/v1/libraries",
		query:  query,
		out:    &response,
	}); err != nil {
		return usecase.Page[domain.Library]{}, err
	}

	return toPage(response, libraryResponse.toDomain), nil
}

func (c *LibraryClient) GetLibrary(ctx context.Context, libraryUID uuid.UUID) (*domain.Library, error) {
	var response libraryResponse
	if err := c.transport.do(ctx, call{
		method:   http.MethodGet,
		path:     "/api/v1/libraries/" + libraryUID.String(),
		out:      &response,
		statuses: map[int]error{http.StatusNotFound: domain.ErrLibraryNotFound},
	}); err != nil {
		return nil, err
	}

	library := response.toDomain()
	return &library, nil
}

func (c *LibraryClient) ListBooks(
	ctx context.Context,
	libraryUID uuid.UUID,
	page, size int,
	showAll bool,
) (usecase.Page[domain.LibraryBook], error) {
	query := paging(page, size)
	query.Set("showAll", strconv.FormatBool(showAll))

	var response pageResponse[libraryBookResponse]
	if err := c.transport.do(ctx, call{
		method:   http.MethodGet,
		path:     "/api/v1/libraries/" + libraryUID.String() + "/books",
		query:    query,
		out:      &response,
		statuses: map[int]error{http.StatusNotFound: domain.ErrLibraryNotFound},
	}); err != nil {
		return usecase.Page[domain.LibraryBook]{}, err
	}

	return toPage(response, libraryBookResponse.toDomain), nil
}

func (c *LibraryClient) GetBook(ctx context.Context, bookUID uuid.UUID) (*domain.Book, error) {
	var response bookResponse
	if err := c.transport.do(ctx, call{
		method:   http.MethodGet,
		path:     "/api/v1/books/" + bookUID.String(),
		out:      &response,
		statuses: map[int]error{http.StatusNotFound: domain.ErrBookNotFound},
	}); err != nil {
		return nil, err
	}

	book := response.toDomain()
	return &book, nil
}

func (c *LibraryClient) TakeBook(ctx context.Context, libraryUID, bookUID uuid.UUID, inQueue bool) (*domain.BookCopy, error) {
	var response bookCopyResponse
	if err := doOrQueue(ctx, c.transport, c.retryer, inQueue, call{
		method: http.MethodPost,
		path:   copyPath(libraryUID, bookUID, "take"),
		out:    &response,
		statuses: map[int]error{
			http.StatusNotFound: domain.ErrBookNotFound,
			http.StatusConflict: domain.ErrNoAvailableCopies,
		},
	}); err != nil {
		return nil, err
	}

	taken := response.toDomain()
	return &taken, nil
}

func (c *LibraryClient) ReturnBook(
	ctx context.Context,
	libraryUID, bookUID uuid.UUID,
	condition domain.BookCondition,
	inQueue bool,
) (*domain.BookCopy, error) {
	var response bookCopyResponse
	if err := doOrQueue(ctx, c.transport, c.retryer, inQueue, call{
		method:   http.MethodPost,
		path:     copyPath(libraryUID, bookUID, "return"),
		body:     returnBookRequest{Condition: string(condition)},
		out:      &response,
		statuses: map[int]error{http.StatusNotFound: domain.ErrBookNotFound},
	}); err != nil {
		return nil, err
	}

	shelved := response.toDomain()
	return &shelved, nil
}

func copyPath(libraryUID, bookUID uuid.UUID, action string) string {
	return "/api/v1/libraries/" + libraryUID.String() + "/books/" + bookUID.String() + "/" + action
}

type libraryResponse struct {
	LibraryUID uuid.UUID `json:"libraryUid"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	City       string    `json:"city"`
}

func (r libraryResponse) toDomain() domain.Library {
	return domain.Library{
		LibraryUID: r.LibraryUID,
		Name:       r.Name,
		City:       r.City,
		Address:    r.Address,
	}
}

type bookResponse struct {
	BookUID uuid.UUID `json:"bookUid"`
	Name    string    `json:"name"`
	Author  string    `json:"author"`
	Genre   string    `json:"genre"`
}

func (r bookResponse) toDomain() domain.Book {
	return domain.Book{
		BookUID: r.BookUID,
		Name:    r.Name,
		Author:  r.Author,
		Genre:   r.Genre,
	}
}

type libraryBookResponse struct {
	BookUID        uuid.UUID `json:"bookUid"`
	Name           string    `json:"name"`
	Author         string    `json:"author"`
	Genre          string    `json:"genre"`
	Condition      string    `json:"condition"`
	AvailableCount int       `json:"availableCount"`
}

func (r libraryBookResponse) toDomain() domain.LibraryBook {
	return domain.LibraryBook{
		Book: domain.Book{
			BookUID: r.BookUID,
			Name:    r.Name,
			Author:  r.Author,
			Genre:   r.Genre,
		},
		Condition:      domain.BookCondition(r.Condition),
		AvailableCount: r.AvailableCount,
	}
}

type bookCopyResponse struct {
	Condition      string `json:"condition"`
	AvailableCount int    `json:"availableCount"`
}

func (r bookCopyResponse) toDomain() domain.BookCopy {
	return domain.BookCopy{
		Condition:      domain.BookCondition(r.Condition),
		AvailableCount: r.AvailableCount,
	}
}

type returnBookRequest struct {
	Condition string `json:"condition"`
}
