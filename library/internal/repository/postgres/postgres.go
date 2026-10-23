package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

const uniqueViolation = "23505"

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func get[T any](ctx context.Context, q querier, statement sq.Sqlizer, scan pgx.RowToFunc[T]) (T, error) {
	rows, err := query(ctx, q, statement)
	if err != nil {
		var zero T
		return zero, err
	}
	return pgx.CollectExactlyOneRow(rows, scan)
}

func list[T any](ctx context.Context, q querier, statement sq.Sqlizer, scan pgx.RowToFunc[T]) ([]T, error) {
	rows, err := query(ctx, q, statement)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scan)
}

func exec(ctx context.Context, q querier, statement sq.Sqlizer) error {
	sql, args, err := statement.ToSql()
	if err != nil {
		return fmt.Errorf("build statement: %w", err)
	}

	_, err = q.Exec(ctx, sql, args...)
	return err
}

func query(ctx context.Context, q querier, statement sq.Sqlizer) (pgx.Rows, error) {
	sql, args, err := statement.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return q.Query(ctx, sql, args...)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

func page(statement sq.SelectBuilder, limit, offset int) sq.SelectBuilder {
	return statement.Limit(uint64(max(limit, 0))).Offset(uint64(max(offset, 0)))
}
