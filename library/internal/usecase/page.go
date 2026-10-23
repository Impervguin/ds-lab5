package usecase

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type Page[T any] struct {
	Page          int
	PageSize      int
	TotalElements int
	Items         []T
}

func normalizePaging(page, size int) (normalizedPage, normalizedSize, limit, offset int) {
	if page < DefaultPage {
		page = DefaultPage
	}
	switch {
	case size <= 0:
		size = DefaultPageSize
	case size > MaxPageSize:
		size = MaxPageSize
	}
	return page, size, size, (page - 1) * size
}
