INSERT INTO payment_crypto (tx_hash, wallet_address, amount_btc, confirmations, status_code)
VALUES
    ('0x1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7', '0x1234567890abcdef1234567890abcdef12345678', 0.00150000, 6, 'done'),
    ('0x2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8', '0xabcdef1234567890abcdef1234567890abcdef12', 0.00080000, 3, 'pend'),
    ('0x3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9', '0x9876543210fedcba9876543210fedcba98765432', 0.02300000, 0, 'pend'),
    ('0x4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0', '0xabcdefabcdefabcdefabcdefabcdefabcdefabcd', 0.00010000, 12, 'done'),
    ('0x5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1', '0x1234567890abcdef1234567890abcdef12345679', 0.05000000, 1, 'fail')
ON CONFLICT (tx_hash) 
DO UPDATE SET 
    wallet_address = EXCLUDED.wallet_address,
    amount_btc = EXCLUDED.amount_btc,
    confirmations = EXCLUDED.confirmations,
    status_code = EXCLUDED.status_code,
    updated_at = CURRENT_TIMESTAMP;