package db

import (
	"database/sql"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"payment_etl_pipeline/internal/config"
	"payment_etl_pipeline/internal/source"
)

type GormDBs struct {
	Cards   *gorm.DB
	Wallets *gorm.DB
	Crypto  *gorm.DB
	Meta    *gorm.DB
}

func NewGormDBs(postgresCfg config.PostgresConfig) (*GormDBs, error) {
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	cards, err := openGorm(postgresCfg.CardsDSN, cfg, source.Cards.String())
	if err != nil {
		return nil, err
	}

	wallets, err := openGorm(postgresCfg.WalletsDSN, cfg, source.Wallets.String())
	if err != nil {
		return nil, err
	}

	crypto, err := openGorm(postgresCfg.CryptoDSN, cfg, source.Crypto.String())
	if err != nil {
		return nil, err
	}

	meta, err := openGorm(postgresCfg.MetaDSN, cfg, source.Meta.String())
	if err != nil {
		return nil, err
	}

	return &GormDBs{
		Cards:   cards,
		Wallets: wallets,
		Crypto:  crypto,
		Meta:    meta,
	}, nil
}

func (dbs *GormDBs) Close() error {
	var errs []error

	for _, pair := range []struct {
		db   *gorm.DB
		name string
	}{
		{dbs.Cards, source.Cards.String()},
		{dbs.Wallets, source.Wallets.String()},
		{dbs.Crypto, source.Crypto.String()},
		{dbs.Meta, source.Meta.String()},
	} {
		if err := closeGormDB(pair.db, pair.name); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("closing connections: %v", errs)
	}

	return nil
}

func (dbs *GormDBs) CardsSQL() (*sql.DB, error)   { return toSQL(dbs.Cards) }
func (dbs *GormDBs) WalletsSQL() (*sql.DB, error) { return toSQL(dbs.Wallets) }
func (dbs *GormDBs) CryptoSQL() (*sql.DB, error)  { return toSQL(dbs.Crypto) }
func (dbs *GormDBs) MetaSQL() (*sql.DB, error)    { return toSQL(dbs.Meta) }

func toSQL(gormDB *gorm.DB) (*sql.DB, error) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get *sql.DB: %w", err)
	}
	return sqlDB, nil
}

func openGorm(dsn string, cfg *gorm.Config, name string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", name, err)
	}

	return db, nil
}

func closeGormDB(gormDB *gorm.DB, name string) error {
	if gormDB == nil {
		return nil
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("%s get DB: %w", name, err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("%s close: %w", name, err)
	}

	return nil
}
