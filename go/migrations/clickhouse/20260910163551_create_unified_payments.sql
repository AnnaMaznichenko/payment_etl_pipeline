-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS default.unified_payments (
    source_system String,
    external_id String,
    user_identifier String,
    amount Float64,
    currency String,
    status String,
    event_at DateTime('UTC'),
    raw_status_original String,
    batch_id UUID,
    ingested_at DateTime('UTC') DEFAULT now()
) ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (source_system, external_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS default.unified_payments;
-- +goose StatementEnd
