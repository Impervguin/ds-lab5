package dto

import (
	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type LibraryResponse struct {
	LibraryUID string `json:"libraryUid"`
	Name       string `json:"name,omitempty"`
	Address    string `json:"address,omitempty"`
	City       string `json:"city,omitempty"`
}

type LibraryPaginationResponse struct {
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
	TotalElements int               `json:"totalElements"`
	Items         []LibraryResponse `json:"items"`
}

func NewLibraryResponse(library domain.Library) LibraryResponse {
	return LibraryResponse{
		LibraryUID: library.LibraryUID.String(),
		Name:       library.Name,
		Address:    library.Address,
		City:       library.City,
	}
}

func NewLibraryPaginationResponse(page usecase.Page[domain.Library]) LibraryPaginationResponse {
	items := make([]LibraryResponse, 0, len(page.Items))
	for _, library := range page.Items {
		items = append(items, NewLibraryResponse(library))
	}

	return LibraryPaginationResponse{
		Page:          page.Page,
		PageSize:      page.PageSize,
		TotalElements: page.TotalElements,
		Items:         items,
	}
}
