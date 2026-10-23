package dto

import "github.com/Impervguin/ds-lab2/library/internal/domain"

type BookResponse struct {
	BookUID string `json:"bookUid"`
	Name    string `json:"name"`
	Author  string `json:"author"`
	Genre   string `json:"genre"`
}

func NewBookResponse(book domain.Book) BookResponse {
	return BookResponse{
		BookUID: book.BookUID.String(),
		Name:    book.Name,
		Author:  book.Author,
		Genre:   book.Genre,
	}
}
