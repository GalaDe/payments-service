-- sql/schema.sql

CREATE TABLE plaid_tokens (
    user_id TEXT PRIMARY KEY,
    access_token TEXT NOT NULL,
    account_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE stripe_customers (
    user_id              TEXT PRIMARY KEY,
    stripe_customer_id   TEXT NOT NULL,
    email                TEXT,
    default_payment_id   TEXT,
    payment_method_type  TEXT, -- e.g., "us_bank_account", "card", etc.
    bank_last4           TEXT, -- last 4 digits of bank account/card
    bank_name            TEXT, -- optional: extracted from Stripe or Plaid
    is_verified          BOOLEAN DEFAULT FALSE,
    created_at           TIMESTAMP DEFAULT NOW(),
    updated_at           TIMESTAMP DEFAULT NOW()
);


CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             TEXT NOT NULL,
    amount              BIGINT NOT NULL, -- in cents
    currency            TEXT NOT NULL DEFAULT 'usd',
    plaid_account_id    TEXT,
    plaid_item_id       TEXT,
    stripe_customer_id  TEXT,
    stripe_payment_id   TEXT,
    status              TEXT NOT NULL DEFAULT 'pending', -- pending, succeeded, failed, canceled
    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW()
);


