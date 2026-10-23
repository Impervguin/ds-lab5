package usecase

const (
	DefaultPage     = 1
	DefaultPageSize = 10
)

type Page[T any] struct {
	Page          int
	PageSize      int
	TotalElements int
	Items         []T
}
