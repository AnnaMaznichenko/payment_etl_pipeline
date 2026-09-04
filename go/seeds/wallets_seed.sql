INSERT INTO payment_wallets (operation_id, user_phone, amount_rub, commission_rub, state)
VALUES
    ('op-1001', '+79001234567', 1500.50, 15.00, 1),
    ('op-1002', '+79007654321', 250.00, 0.00, 0),
    ('op-1003', '+79111234567', 8000.00, 80.00, 2),
    ('op-1004', '+79229876543', 120.75, 1.25, 1),
    ('op-1005', '+79331234567', 4500.00, 45.00, 0)
ON CONFLICT (operation_id) 
DO UPDATE SET 
    user_phone = EXCLUDED.user_phone,
    amount_rub = EXCLUDED.amount_rub,
    commission_rub = EXCLUDED.commission_rub,
    state = EXCLUDED.state,
    updated_at = CURRENT_TIMESTAMP;