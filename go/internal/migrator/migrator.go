package migrator

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	"github.com/pressly/goose/v3"

	"payment_etl_pipeline/internal/source"
)

const (
	DialectPostgres   = "postgres"
	DialectClickHouse = "clickhouse"
)

type MigrationTarget struct {
	Name string
	DB   *sql.DB
	Sub  string
}

func Run(baseDir string, dialect string, targets []MigrationTarget) error {
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("set dialect %s: %w", dialect, err)
	}

	for _, t := range targets {
		dir := filepath.Join(baseDir, t.Sub)
		log.Printf("[migrator] %s: applying migrations from %s", t.Name, dir)
		if err := goose.Up(t.DB, dir); err != nil {
			return fmt.Errorf("goose up %s (%s): %w", t.Name, dir, err)
		}
	}

	return nil
}

func PostgresTarget(src source.Source, db *sql.DB) MigrationTarget {
	return MigrationTarget{
		Name: src.String(),
		DB:   db,
		Sub:  filepath.Join("postgres", src.String()),
	}
}
