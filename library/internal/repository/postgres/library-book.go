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

type LibraryBookRepository struct {
	pool *pgxpool.Pool
}

func NewLibraryBookRepository(pool *pgxpool.Pool) *LibraryBookRepository {
	return &LibraryBookRepository{pool: pool}
}

var _ domain.LibraryBookRepository = (*LibraryBookRepository)(nil)

const lockHoldingQuery = `SELECT pg_advisory_xact_lock($1, $2)`

func libraryBookSelect() sq.SelectBuilder {
	return builder.
		Select(append(bookColumns, "lb.library_id", "lb.condition", "lb.available_count")...).
		From("library_books lb").
		Join("library l ON l.id = lb.library_id").
		Join("books b ON b.id = lb.book_id")
}

func scanLibraryBook(row pgx.CollectableRow) (domain.LibraryBook, error) {
	var libraryBook domain.LibraryBook
	return libraryBook, row.Scan(
		&libraryBook.Book.ID, &libraryBook.Book.BookUID, &libraryBook.Book.Name,
		&libraryBook.Book.Author, &libraryBook.Book.Genre,
		&libraryBook.LibraryID, &libraryBook.Condition, &libraryBook.AvailableCount,
	)
}

func (r *LibraryBookRepository) ListByLibrary(
	ctx context.Context,
	libraryUID uuid.UUID,
	limit, offset int,
	includeEmpty bool,
) ([]domain.LibraryBook, int, error) {
	held := sq.And{sq.Eq{"l.library_uid": libraryUID}}
	if !includeEmpty {
		held = append(held, sq.Gt{"lb.available_count": 0})
	}

	counter := builder.Select("count(*)").
		From("library_books lb").
		Join("library l ON l.id = lb.library_id").
		Where(held)
	total, err := get(ctx, r.pool, counter, pgx.RowTo[int])
	if err != nil {
		return nil, 0, fmt.Errorf("count books of library %s: %w", libraryUID, err)
	}

	listing := libraryBookSelect().Where(held).OrderBy("b.name", "b.id", "lb.condition")
	books, err := list(ctx, r.pool, page(listing, limit, offset), scanLibraryBook)
	if err != nil {
		return nil, 0, fmt.Errorf("list books of library %s: %w", libraryUID, err)
	}
	return books, total, nil
}

func (r *LibraryBookRepository) Update(
	ctx context.Context,
	libraryUID, bookUID uuid.UUID,
	updFunc domain.LibraryBooksUpdateFunc,
) ([]domain.LibraryBook, error) {
	var updated []domain.LibraryBook

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		libraryID, bookID, err := r.resolve(ctx, tx, libraryUID, bookUID)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, lockHoldingQuery, libraryID, bookID); err != nil {
			return fmt.Errorf("lock book %s in library %s: %w", bookUID, libraryUID, err)
		}

		held, err := list(ctx, tx, libraryBookSelect().
			Where(sq.Eq{"lb.library_id": libraryID, "lb.book_id": bookID}), scanLibraryBook)
		if err != nil {
			return fmt.Errorf("load book %s in library %s: %w", bookUID, libraryUID, err)
		}
		if len(held) == 0 {
			return domain.ErrBookNotFound
		}

		counts := make(map[domain.BookCondition]int, len(held))
		for _, row := range held {
			counts[row.Condition] = row.AvailableCount
		}

		changed, err := updFunc(ctx, held)
		if err != nil {
			return err
		}

		if err := r.save(ctx, tx, libraryID, bookID, counts, changed); err != nil {
			return fmt.Errorf("save book %s in library %s: %w", bookUID, libraryUID, err)
		}

		updated = changed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *LibraryBookRepository) resolve(ctx context.Context, tx pgx.Tx, libraryUID, bookUID uuid.UUID) (int64, int64, error) {
	libraryID, err := get(ctx, tx, builder.Select("id").From("library").Where(sq.Eq{"library_uid": libraryUID}), pgx.RowTo[int64])
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, 0, domain.ErrLibraryNotFound
	case err != nil:
		return 0, 0, fmt.Errorf("get library %s: %w", libraryUID, err)
	}

	bookID, err := get(ctx, tx, builder.Select("id").From("books").Where(sq.Eq{"book_uid": bookUID}), pgx.RowTo[int64])
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, 0, domain.ErrBookNotFound
	case err != nil:
		return 0, 0, fmt.Errorf("get book %s: %w", bookUID, err)
	}

	return libraryID, bookID, nil
}

func (r *LibraryBookRepository) save(
	ctx context.Context,
	tx pgx.Tx,
	libraryID, bookID int64,
	counts map[domain.BookCondition]int,
	changed []domain.LibraryBook,
) error {
	for _, row := range changed {
		count, held := counts[row.Condition]
		switch {
		case !held:
			err := exec(ctx, tx, builder.
				Insert("library_books").
				Columns("library_id", "book_id", "condition", "available_count").
				Values(libraryID, bookID, row.Condition, row.AvailableCount))
			if err != nil {
				return err
			}
		case count != row.AvailableCount:
			err := exec(ctx, tx, builder.
				Update("library_books").
				Set("available_count", row.AvailableCount).
				Where(sq.Eq{"library_id": libraryID, "book_id": bookID, "condition": row.Condition}))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
