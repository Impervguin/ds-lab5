package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

func Run(db *sql.DB) error {
	goose.SetBaseFS(migrationFiles)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, "sql"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
