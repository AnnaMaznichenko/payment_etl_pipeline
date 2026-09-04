-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS payment_wallets (
    id SERIAL PRIMARY KEY,
    operation_id VARCHAR(50) UNIQUE NOT NULL,
    user_phone VARCHAR(15) NOT NULL,
    amount_rub DECIMAL(12,2) NOT NULL,
    commission_rub DECIMAL(10,2) DEFAULT 0,
    state SMALLINT NOT NULL,
    processed_dt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payment_wallets CASCADE;
-- +goose StatementEnd