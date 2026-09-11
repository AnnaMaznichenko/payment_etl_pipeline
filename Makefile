include deployments/.env

# Запуск приложения в правильном порядке
.PHONY: full-up
full-up: docker-down run

# Запуск контейнеров с БД
.PHONY: infra-up
infra-up:
	docker-compose -f deployments/docker-compose.yml up -d postgres-cards postgres-wallets postgres-crypto postgres-meta clickhouse

# Запуск приложения в Docker
.PHONY: run
run:
	docker-compose -f deployments/docker-compose.yml up -d --build

.PHONY: docker-down
docker-down:
	docker-compose -f deployments/docker-compose.yml down

.PHONY: docker-logs
docker-logs:
	docker-compose -f deployments/docker-compose.yml logs -f

# Запустить только приложение (без пересборки)
.PHONY: run-app
run-app:
	docker-compose -f deployments/docker-compose.yml up -d app

# Миграции postgres (через goose)

POSTGRES_USER ?= user
POSTGRES_CARDS_DB ?= cards_db
POSTGRES_WALLETS_DB ?= wallets_db
POSTGRES_CRYPTO_DB ?= crypto_db
POSTGRES_META_DB ?= meta_db

GOOSE_DRIVER=postgres
GOOSE_CARDS_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_CARDS_DB)?sslmode=disable
GOOSE_WALLETS_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5433/$(POSTGRES_WALLETS_DB)?sslmode=disable
GOOSE_CRYPTO_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5434/$(POSTGRES_CRYPTO_DB)?sslmode=disable
GOOSE_META_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5435/$(POSTGRES_META_DB)?sslmode=disable

.PHONY: migrate-cards-up
migrate-cards-up:
	goose -dir ./go/migrations/postgres/cards $(GOOSE_DRIVER) "$(GOOSE_CARDS_DSN)" up

.PHONY: migrate-wallets-up
migrate-wallets-up:
	goose -dir ./go/migrations/postgres/wallets $(GOOSE_DRIVER) "$(GOOSE_WALLETS_DSN)" up

.PHONY: migrate-crypto-up
migrate-crypto-up:
	goose -dir ./go/migrations/postgres/crypto $(GOOSE_DRIVER) "$(GOOSE_CRYPTO_DSN)" up

.PHONY: migrate-meta-up
migrate-meta-up:
	goose -dir ./go/migrations/postgres/meta $(GOOSE_DRIVER) "$(GOOSE_META_DSN)" up

# Откат миграций postgres

.PHONY: migrate-cards-down
migrate-cards-down:
	goose -dir ./go/migrations/postgres/cards $(GOOSE_DRIVER) "$(GOOSE_CARDS_DSN)" down

.PHONY: migrate-wallets-down
migrate-wallets-down:
	goose -dir ./go/migrations/postgres/wallets $(GOOSE_DRIVER) "$(GOOSE_WALLETS_DSN)" down

.PHONY: migrate-crypto-down
migrate-crypto-down:
	goose -dir ./go/migrations/postgres/crypto $(GOOSE_DRIVER) "$(GOOSE_CRYPTO_DSN)" down

.PHONY: migrate-meta-down
migrate-meta-down:
	goose -dir ./go/migrations/postgres/meta $(GOOSE_DRIVER) "$(GOOSE_META_DSN)" down

# Миграции ClickHouse (через goose)

CLICKHOUSE_GOOSE_DSN=clickhouse://$(CLICKHOUSE_USER):$(CLICKHOUSE_PASSWORD)@localhost:9000/default

.PHONY: migrate-clickhouse-up
migrate-clickhouse-up:
	goose -dir ./go/migrations/clickhouse clickhouse "$(CLICKHOUSE_GOOSE_DSN)" up

.PHONY: migrate-clickhouse-down
migrate-clickhouse-down:
	goose -dir ./go/migrations/clickhouse clickhouse "$(CLICKHOUSE_GOOSE_DSN)" down

# Запуск всех миграций
.PHONY: migrate-all-up
migrate-all-up: migrate-cards-up migrate-wallets-up migrate-crypto-up migrate-meta-up migrate-clickhouse-up

# Откат всех миграций
.PHONY: migrate-all-down
migrate-all-down: migrate-cards-down migrate-wallets-down migrate-crypto-down migrate-meta-down migrate-clickhouse-down

# Seed-данные
.PHONY: seed-cards
seed-cards:
	docker exec -i postgres-cards psql -U $(POSTGRES_USER) -d $(POSTGRES_CARDS_DB) < ./go/seeds/cards_seed.sql

.PHONY: seed-wallets
seed-wallets:
	docker exec -i postgres-wallets psql -U $(POSTGRES_USER) -d $(POSTGRES_WALLETS_DB) < ./go/seeds/wallets_seed.sql

.PHONY: seed-crypto
seed-crypto:
	docker exec -i postgres-crypto psql -U $(POSTGRES_USER) -d $(POSTGRES_CRYPTO_DB) < ./go/seeds/crypto_seed.sql

.PHONY: seed-all
seed-all: seed-cards seed-wallets seed-crypto