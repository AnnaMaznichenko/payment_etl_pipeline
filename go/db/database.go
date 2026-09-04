package db

import (
	"database/sql"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

type GormDBs struct {
	Cards   *gorm.DB
	Wallets *gorm.DB
	Crypto  *gorm.DB
	Meta    *gorm.DB
}

func NewGormDBs(cardsDSN, walletsDSN, cryptoDSN, metaDSN string) (*GormDBs, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	var err error
	dbs := &GormDBs{}

	// cards_db
	dbs.Cards, err = gorm.Open(postgres.Open(cardsDSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to cards: %w", err)
	}

	// wallets_db
	dbs.Wallets, err = gorm.Open(postgres.Open(walletsDSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to wallets: %w", err)
	}

	// crypto_db
	dbs.Crypto, err = gorm.Open(postgres.Open(cryptoDSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to crypto: %w", err)
	}

	// meta_db
	dbs.Meta, err = gorm.Open(postgres.Open(metaDSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to meta: %w", err)
	}

	log.Println("All GORM connections established and migrations applied")

	return dbs, nil
}

func (dbs *GormDBs) Close() error {
	var errs []error

	if err := closeGormDB(dbs.Cards, "cards"); err != nil {
		errs = append(errs, err)
	}
	if err := closeGormDB(dbs.Wallets, "wallets"); err != nil {
		errs = append(errs, err)
	}
	if err := closeGormDB(dbs.Crypto, "crypto"); err != nil {
		errs = append(errs, err)
	}
	if err := closeGormDB(dbs.Meta, "meta"); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("closing connections: %v", errs)
	}

	return nil
}

func closeGormDB(db *gorm.DB, name string) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("%s get DB: %w", name, err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("%s close: %w", name, err)
	}

	return nil
}

type ClickHouseDB struct {
	DB *sql.DB
}

func NewClickHouseDB(ClickhouseDSN string) (*ClickHouseDB, error) {
	db, err := sql.Open("clickhouse", ClickhouseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open ClickHouse connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &ClickHouseDB{DB: db}, nil
}

func (db *ClickHouseDB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}

	return nil
}
