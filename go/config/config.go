package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	CardsDSN      string
	WalletsDSN    string
	CryptoDSN     string
	MetaDSN       string
	ClickhouseDSN string

	BatchSize       int
	Timeout         time.Duration
	PollInterval    time.Duration
	ShutdownTimeout time.Duration
}

func Load() *Config {
	cfg := &Config{
		CardsDSN:        getEnvString("CARDS_DSN", ""),
		WalletsDSN:      getEnvString("WALLETS_DSN", ""),
		CryptoDSN:       getEnvString("CRYPTO_DSN", ""),
		MetaDSN:         getEnvString("META_DSN", ""),
		ClickhouseDSN:   getEnvString("CLICKHOUSE_DSN", ""),
		BatchSize:       getEnvInt("BATCH_SIZE", 100),
		Timeout:         getEnvDuration("TIMEOUT", 20*time.Second),
		PollInterval:    getEnvDuration("POLL_INTERVAL", 5*time.Second),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
	}

	return cfg
}

func getEnvString(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	log.Printf("WARNING: %s is not set, using default %q", key, defaultVal)
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
		log.Printf("WARNING: invalid %s value %q, using default %d", key, val, defaultVal)
	}

	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		log.Printf("WARNING: invalid %s value %q, using default %v", key, val, defaultVal)
	}

	return defaultVal
}
