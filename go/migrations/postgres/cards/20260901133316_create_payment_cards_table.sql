-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS payment_cards (
    id SERIAL PRIMARY KEY,
    transaction_id VARCHAR(36) UNIQUE NOT NULL,
    card_number_masked VARCHAR(19),
    amount_kopecks BIGINT NOT NULL,
    currency_code VARCHAR(3) DEFAULT 'RUB',
    payment_status VARCHAR(20),
    gateway_response TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payment_cards CASCADE;
-- +goose StatementEnd
