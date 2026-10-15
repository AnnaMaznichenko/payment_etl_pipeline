package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testCardsDB   *gorm.DB
	testWalletsDB *gorm.DB
	testCryptoDB  *gorm.DB
	testMetaDB    *gorm.DB
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	type dbSetup struct {
		target    **gorm.DB
		name      string
		dir       string
		container testcontainers.Container
	}

	setups := []*dbSetup{
		{target: &testCardsDB, name: "cards_db", dir: "../../migrations/postgres/cards"},
		{target: &testWalletsDB, name: "wallets_db", dir: "../../migrations/postgres/wallets"},
		{target: &testCryptoDB, name: "crypto_db", dir: "../../migrations/postgres/crypto"},
		{target: &testMetaDB, name: "meta_db", dir: "../../migrations/postgres/meta"},
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose dialect: %v", err)
	}

	for _, s := range setups {
		*s.target, s.container = startPostgres(ctx, s.name, s.dir)
	}

	code := m.Run()

	for _, s := range setups {
		if s.container != nil {
			_ = s.container.Terminate(context.Background())
		}
	}

	cancel()
	os.Exit(code)
}

func startPostgres(ctx context.Context, dbName, migrationsDir string) (*gorm.DB, testcontainers.Container) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       dbName,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("start %s: %v", dbName, err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("host %s: %v", dbName, err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("port %s: %v", dbName, err)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/%s?sslmode=disable&TimeZone=UTC",
		host, port.Port(), dbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("gorm open %s: %v", dbName, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("db.DB() %s: %v", dbName, err)
	}

	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		log.Fatalf("migrations %s: %v", dbName, err)
	}

	return db, container
}
