package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"payment_etl_pipeline/batcher"
	"payment_etl_pipeline/config"
	"payment_etl_pipeline/db"
	"payment_etl_pipeline/extractor"
	"payment_etl_pipeline/loader"
	"payment_etl_pipeline/normalizer"
	"payment_etl_pipeline/orchestrator"
	"payment_etl_pipeline/repository"
)

func main() {
	cfg := config.Load()

	gormDBs, err := db.NewGormDBs(cfg.CardsDSN, cfg.WalletsDSN, cfg.CryptoDSN, cfg.MetaDSN)
	if err != nil {
		log.Fatalf("GORM init: %v", err)
	}
	defer gormDBs.Close()

	chDB, err := db.NewClickHouseDB(cfg.ClickhouseDSN)
	if err != nil {
		log.Fatalf("ClickHouse init: %v", err)
	}
	defer chDB.Close()

	log.Println("All databases ready")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cardRepo := repository.NewCardRepository()
	cryptoRepo := repository.NewCryptoRepository()
	walletRepo := repository.NewWalletRepository()

	cardExtractor := extractor.NewCardExtractor(cardRepo)
	cryptoExtractor := extractor.NewCCryptoExtractor(cryptoRepo)
	walletExtractor := extractor.NewWalletExtractor(walletRepo)

	cardNormalizator := normalizer.NewCardsNormalizer()
	cryptoNormalizator := normalizer.NewCryptoNormalizer()
	walletNormalizator := normalizer.NewWalletsNormalizer()

	batcher := batcher.NewBatcher(cfg.BatchSize, cfg.Timeout)

	loader := loader.NewLoader()

	checkpointRepo := repository.NewCheckpointRepository()

	o := orchestrator.NewOrchestrator(
		cardExtractor,
		cryptoExtractor,
		walletExtractor,
		cardNormalizator,
		cryptoNormalizator,
		walletNormalizator,
		batcher,
		loader,
		checkpointRepo,
		cfg.PollInterval,
	)

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

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Orchestrator finished gracefully")
	case <-time.After(cfg.ShutdownTimeout):
		log.Println("Orchestrator did not finish in time, forcing exit")
	}
}
