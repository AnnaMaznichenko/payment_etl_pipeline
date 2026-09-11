package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"payment_etl_pipeline/internal/batcher"
	"payment_etl_pipeline/internal/config"
	"payment_etl_pipeline/internal/db"
	"payment_etl_pipeline/internal/extractor"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/loader"
	"payment_etl_pipeline/internal/migrator"
	"payment_etl_pipeline/internal/normalizer"
	"payment_etl_pipeline/internal/orchestrator"
	"payment_etl_pipeline/internal/repository"
	"payment_etl_pipeline/internal/source"
)

func main() {
	cfg := config.Load()

	gormDBs, err := db.NewGormDBs(cfg.Postgres)
	if err != nil {
		log.Fatalf("GORM init: %v", err)
	}
	defer func() {
		if err := gormDBs.Close(); err != nil {
			log.Printf("error closing GORM connections: %v", err)
		}
	}()

	chDB, err := db.NewClickHouseDB(cfg.ClickHouse)
	if err != nil {
		log.Fatalf("ClickHouse init: %v", err)
	}
	defer func() {
		if err := chDB.Close(); err != nil {
			log.Printf("error closing ClickHouse connections: %v", err)
		}
	}()

	err = runMigrations(cfg, gormDBs, chDB)
	if err != nil {
		log.Fatalf("Migrations failed: %v", err)
	}

	log.Println("All databases ready")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cardRepo := repository.NewCardRepository(gormDBs.Cards)
	cryptoRepo := repository.NewCryptoRepository(gormDBs.Crypto)
	walletRepo := repository.NewWalletRepository(gormDBs.Wallets)

	cardExtractor := extractor.NewCardExtractor(cardRepo, cfg.Pipeline.PageSize)
	cryptoExtractor := extractor.NewCryptoExtractor(cryptoRepo, cfg.Pipeline.PageSize)
	walletExtractor := extractor.NewWalletExtractor(walletRepo, cfg.Pipeline.PageSize)

	cardNormalizator := normalizer.NewCardsNormalizer()
	cryptoNormalizator := normalizer.NewCryptoNormalizer()
	walletNormalizator := normalizer.NewWalletsNormalizer()

	batcher := batcher.NewBatcher(cfg.Pipeline.BatchSize, cfg.Pipeline.Timeout)
	loader := loader.NewLoader(chDB)
	checkpointRepo := repository.NewCheckpointRepository(gormDBs.Meta)

	o := orchestrator.NewOrchestrator(
		orchestrator.Extractors{
			Cards:   cardExtractor,
			Crypto:  cryptoExtractor,
			Wallets: walletExtractor,
		},
		orchestrator.Normalizers{
			Cards:   cardNormalizator,
			Crypto:  cryptoNormalizator,
			Wallets: walletNormalizator,
		},
		batcher,
		loader,
		checkpointRepo,
		cfg.Pipeline.PollInterval,
	)

	runWithGracefulShutdown(ctx, cancel, o, cfg.Pipeline.ShutdownTimeout)
}

func runMigrations(cfg *config.Config, gormDBs *db.GormDBs, chDB *db.ClickHouseDB) error {
	// PostgreSQL
	cardsDB, err := gormDBs.CardsSQL()
	if err != nil {
		return err
	}
	cryptoDB, err := gormDBs.CryptoSQL()
	if err != nil {
		return err
	}
	walletsDB, err := gormDBs.WalletsSQL()
	if err != nil {
		return err
	}
	metaDB, err := gormDBs.MetaSQL()
	if err != nil {
		return err
	}

	pgTargets := []migrator.MigrationTarget{
		migrator.PostgresTarget(source.Cards, cardsDB),
		migrator.PostgresTarget(source.Crypto, cryptoDB),
		migrator.PostgresTarget(source.Wallets, walletsDB),
		migrator.PostgresTarget(source.Meta, metaDB),
	}
	if err := migrator.Run(cfg.Migrations.Dir, migrator.DialectPostgres, pgTargets); err != nil {
		return err
	}

	// ClickHouse
	chTargets := []migrator.MigrationTarget{
		{Name: source.ClickHouse.String(), DB: chDB.DB, Sub: "clickhouse"},
	}
	if err := migrator.Run(cfg.Migrations.Dir, migrator.DialectClickHouse, chTargets); err != nil {
		return err
	}

	return nil
}

func runWithGracefulShutdown(ctx context.Context, cancel context.CancelFunc, o interfaces.Orchestrator, shutdownTimeout time.Duration) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := o.Run(ctx); err != nil {
			log.Printf("Orchestrator stopped with error: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Received shutdown signal, cancelling context...")
	cancel()

	// Ждём завершения оркестратора с таймаутом
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Orchestrator finished gracefully")
	case <-time.After(shutdownTimeout):
		log.Println("Orchestrator did not finish in time, forcing exit")
	}
}
