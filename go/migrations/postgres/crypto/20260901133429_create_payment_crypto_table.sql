-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS payment_crypto (
    id SERIAL PRIMARY KEY,
    tx_hash VARCHAR(66) UNIQUE NOT NULL,
    wallet_address VARCHAR(42) NOT NULL,
    amount_btc DECIMAL(16,8) NOT NULL,
    confirmations INTEGER DEFAULT 0,
    status_code VARCHAR(10) DEFAULT 'pend',
    block_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payment_crypto CASCADE;
-- +goose StatementEnd