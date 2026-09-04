-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS etl_checkpoints (
    source_name VARCHAR(20) PRIMARY KEY,
    last_processed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS etl_checkpoints CASCADE;
-- +goose StatementEnd