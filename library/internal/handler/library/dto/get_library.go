package dto

import "github.com/Impervguin/ds-lab2/library/internal/domain"

type LibraryResponse struct {
	LibraryUID string `json:"libraryUid"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	City       string `json:"city"`
}

func NewLibraryResponse(library domain.Library) LibraryResponse {
	return LibraryResponse{
		LibraryUID: library.LibraryUID.String(),
		Name:       library.Name,
		Address:    library.Address,
		City:       library.City,
	}
}
