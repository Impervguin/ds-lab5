package dto

import (
	"github.com/Impervguin/ds-lab2/library/internal/domain"
	"github.com/Impervguin/ds-lab2/library/internal/usecase"
)

type LibraryPaginationResponse struct {
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
	TotalElements int               `json:"totalElements"`
	Items         []LibraryResponse `json:"items"`
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
