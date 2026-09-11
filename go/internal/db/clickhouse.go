package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/ClickHouse/clickhouse-go/v2"

	"payment_etl_pipeline/internal/config"
)

type ClickHouseDB struct {
	Conn clickhouse.Conn
	DB   *sql.DB
}

func NewClickHouseDB(cfg config.ClickHouseConfig) (*ClickHouseDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.PingTimeout)
	defer cancel()

	// нативное соединение для работы приложения
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping native connection: %w", err)
	}

	// SQL-соединение для миграций
	u := url.URL{
		Scheme: "clickhouse",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + cfg.Database,
	}
	dsn := u.String()
	sqlDB, err := sql.Open("clickhouse", dsn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open sql connection: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = conn.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sql connection: %w", err)
	}

	return &ClickHouseDB{
		Conn: conn,
		DB:   sqlDB,
	}, nil
}

func (ch *ClickHouseDB) Close() error {
	var errs []error
	if ch.Conn != nil {
		if err := ch.Conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if ch.DB != nil {
		if err := ch.DB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("closing ClickHouse: %v", errs)
	}

	return nil
}
