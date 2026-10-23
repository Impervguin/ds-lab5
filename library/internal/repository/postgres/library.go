package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Impervguin/ds-lab2/library/internal/domain"
)

type LibraryRepository struct {
	pool *pgxpool.Pool
}

func NewLibraryRepository(pool *pgxpool.Pool) *LibraryRepository {
	return &LibraryRepository{pool: pool}
}

var _ domain.LibraryRepository = (*LibraryRepository)(nil)

func librarySelect() sq.SelectBuilder {
	return builder.Select("id", "library_uid", "name", "city", "address").From("library")
}

func scanLibrary(row pgx.CollectableRow) (domain.Library, error) {
	var library domain.Library
	return library, row.Scan(&library.ID, &library.LibraryUID, &library.Name, &library.City, &library.Address)
}

func (r *LibraryRepository) ListByCity(ctx context.Context, city string, limit, offset int) ([]domain.Library, int, error) {
	inCity := sq.Eq{"city": city}

	total, err := get(ctx, r.pool, builder.Select("count(*)").From("library").Where(inCity), pgx.RowTo[int])
	if err != nil {
		return nil, 0, fmt.Errorf("count libraries in %q: %w", city, err)
	}

	libraries, err := list(ctx, r.pool, page(librarySelect().Where(inCity).OrderBy("name", "id"), limit, offset), scanLibrary)
	if err != nil {
		return nil, 0, fmt.Errorf("list libraries in %q: %w", city, err)
	}
	return libraries, total, nil
}

func (r *LibraryRepository) Get(ctx context.Context, libraryUID uuid.UUID) (*domain.Library, error) {
	library, err := get(ctx, r.pool, librarySelect().Where(sq.Eq{"library_uid": libraryUID}), scanLibrary)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrLibraryNotFound
	case err != nil:
		return nil, fmt.Errorf("get library %s: %w", libraryUID, err)
	}
	return &library, nil
}
