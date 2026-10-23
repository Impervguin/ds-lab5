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

type BookRepository struct {
	pool *pgxpool.Pool
}

func NewBookRepository(pool *pgxpool.Pool) *BookRepository {
	return &BookRepository{pool: pool}
}

var _ domain.BookRepository = (*BookRepository)(nil)

var bookColumns = []string{"b.id", "b.book_uid", "b.name", "COALESCE(b.author, '')", "COALESCE(b.genre, '')"}

func bookSelect() sq.SelectBuilder {
	return builder.Select(bookColumns...).From("books b")
}

func scanBook(row pgx.CollectableRow) (domain.Book, error) {
	var book domain.Book
	return book, row.Scan(&book.ID, &book.BookUID, &book.Name, &book.Author, &book.Genre)
}

func (r *BookRepository) Get(ctx context.Context, bookUID uuid.UUID) (*domain.Book, error) {
	book, err := get(ctx, r.pool, bookSelect().Where(sq.Eq{"b.book_uid": bookUID}), scanBook)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrBookNotFound
	case err != nil:
		return nil, fmt.Errorf("get book %s: %w", bookUID, err)
	}
	return &book, nil
}
