-- name: InsertPayment :exec
INSERT INTO payments (
    user_id, amount, currency, plaid_account_id,
    plaid_item_id, stripe_customer_id, stripe_payment_id, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: UpdatePaymentStatus :exec
UPDATE payments SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1;

-- name: GetAllPayments :many
SELECT * FROM payments ORDER BY created_at DESC;
