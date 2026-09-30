package config

import (
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Postgres   PostgresConfig
	ClickHouse ClickHouseConfig
	Migrations MigrationsConfig
	Pipeline   PipelineConfig
	Generator  GeneratorConfig
}

type PostgresConfig struct {
	CardsDSN   string `env:"CARDS_DSN,required"`
	WalletsDSN string `env:"WALLETS_DSN,required"`
	CryptoDSN  string `env:"CRYPTO_DSN,required"`
	MetaDSN    string `env:"META_DSN,required"`
}

type ClickHouseConfig struct {
	Host        string        `env:"CLICKHOUSE_HOST" envDefault:"localhost"`
	Port        int           `env:"CLICKHOUSE_PORT" envDefault:"9000"`
	User        string        `env:"CLICKHOUSE_USER" envDefault:"default"`
	Password    string        `env:"CLICKHOUSE_PASSWORD"`
	Database    string        `env:"CLICKHOUSE_DATABASE" envDefault:"default"`
	PingTimeout time.Duration `env:"CLICKHOUSE_PING_TIMEOUT" envDefault:"5s"`
}

type MigrationsConfig struct {
	Dir string `env:"MIGRATIONS_DIR" envDefault:"migrations"`
}

type PipelineConfig struct {
	PageSize        int           `env:"PAGE_SIZE" envDefault:"1000"`
	BatchSize       int           `env:"BATCH_SIZE" envDefault:"100"`
	Timeout         time.Duration `env:"TIMEOUT" envDefault:"20s"`
	PollInterval    time.Duration `env:"POLL_INTERVAL" envDefault:"5s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type GeneratorConfig struct {
	Enabled         bool          `env:"GENERATOR_ENABLED" envDefault:"true"`
	CardsInterval   time.Duration `env:"GENERATOR_CARDS_INTERVAL"  envDefault:"3s"`
	WalletsInterval time.Duration `env:"GENERATOR_WALLETS_INTERVAL" envDefault:"5s"`
	CryptoInterval  time.Duration `env:"GENERATOR_CRYPTO_INTERVAL" envDefault:"7s"`
}

func Load() *Config {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Printf("failed to load .env: %v", err)
		}
	}

	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	return cfg
}
