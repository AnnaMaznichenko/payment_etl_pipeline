INSERT INTO payment_cards (transaction_id, card_number_masked, amount_kopecks, currency_code, payment_status, gateway_response)
VALUES
    ('123e4567-e89b-12d3-a456-426614174000', '**** **** **** 1111', 100000, 'RUB', 'succeeded', '{"status":"ok"}'),
    ('223e4567-e89b-12d3-a456-426614174001', '**** **** **** 2222', 250000, 'RUB', 'pending', '{"status":"wait"}'),
    ('323e4567-e89b-12d3-a456-426614174002', '**** **** **** 3333', 500000, 'USD', 'failed', '{"error":"insufficient funds"}')
ON CONFLICT (transaction_id) 
DO UPDATE SET 
    card_number_masked = EXCLUDED.card_number_masked,
    amount_kopecks = EXCLUDED.amount_kopecks,
    currency_code = EXCLUDED.currency_code,
    payment_status = EXCLUDED.payment_status,
    gateway_response = EXCLUDED.gateway_response,
    updated_at = CURRENT_TIMESTAMP;