-- name: GetPlaidTokenByUserID :one
SELECT access_token, account_id, item_id
FROM plaid_tokens
WHERE user_id = $1;

-- name: UpsertPlaidToken :exec
INSERT INTO plaid_tokens (user_id, access_token, account_id, item_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET access_token = EXCLUDED.access_token,
    account_id = EXCLUDED.account_id,
    item_id = EXCLUDED.item_id;


-- name: DeletePlaidToken :exec
DELETE FROM plaid_tokens
WHERE user_id = $1;
