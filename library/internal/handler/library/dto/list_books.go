package dto

import (
	"github.com/Impervguin/ds-lab2/library/internal/domain"
	"github.com/Impervguin/ds-lab2/library/internal/usecase"
)

type LibraryBookResponse struct {
	BookUID        string `json:"bookUid"`
	Name           string `json:"name"`
	Author         string `json:"author"`
	Genre          string `json:"genre"`
	Condition      string `json:"condition"`
	AvailableCount int    `json:"availableCount"`
}

type LibraryBookPaginationResponse struct {
	Page          int                   `json:"page"`
	PageSize      int                   `json:"pageSize"`
	TotalElements int                   `json:"totalElements"`
	Items         []LibraryBookResponse `json:"items"`
}

func NewLibraryBookResponse(book domain.LibraryBook) LibraryBookResponse {
	return LibraryBookResponse{
		BookUID:        book.Book.BookUID.String(),
		Name:           book.Book.Name,
		Author:         book.Book.Author,
		Genre:          book.Book.Genre,
		Condition:      string(book.Condition),
		AvailableCount: book.AvailableCount,
	}
}

func NewLibraryBookPaginationResponse(page usecase.Page[domain.LibraryBook]) LibraryBookPaginationResponse {
	items := make([]LibraryBookResponse, 0, len(page.Items))
	for _, book := range page.Items {
		items = append(items, NewLibraryBookResponse(book))
	}

	return LibraryBookPaginationResponse{
		Page:          page.Page,
		PageSize:      page.PageSize,
		TotalElements: page.TotalElements,
		Items:         items,
	}
}
