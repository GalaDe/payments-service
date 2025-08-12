-- name: InsertStripeCustomer :exec
INSERT INTO stripe_customers (
    user_id,
    stripe_customer_id,
    email,
    default_payment_id,
    payment_method_type,
    bank_last4,
    bank_name,
    is_verified
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (user_id) DO UPDATE
SET
    stripe_customer_id = EXCLUDED.stripe_customer_id,
    email = EXCLUDED.email,
    default_payment_id = EXCLUDED.default_payment_id,
    payment_method_type = EXCLUDED.payment_method_type,
    bank_last4 = EXCLUDED.bank_last4,
    bank_name = EXCLUDED.bank_name,
    is_verified = EXCLUDED.is_verified,
    updated_at = NOW();

-- name: GetStripeCustomerByUserID :one
SELECT * FROM stripe_customers
WHERE user_id = $1;

-- name: UpdateStripeCustomerDefaultPayment :exec
UPDATE stripe_customers
SET
    default_payment_id = $2,
    updated_at = NOW()
WHERE user_id = $1;

-- name: DeleteStripeCustomer :exec
DELETE FROM stripe_customers
WHERE user_id = $1;
