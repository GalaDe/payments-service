package domain

import (
	"time"

	"github.com/guregu/null"
)

type PlaidToken struct {
	UserID      string
	AccessToken string
	AccountID   string
	ItemID      string
}

type StripeCustomer struct {
	UserID            string      `json:"user_id"`             // Internal user ID
	StripeCustomerID  string      `json:"stripe_customer_id"`  // Stripe customer ID
	Email             null.String `json:"email"`               // Optional: customer email
	DefaultPaymentID  null.String `json:"default_payment_id"`  // Default payment method ID
	PaymentMethodType null.String `json:"payment_method_type"` // e.g., "us_bank_account", "card"
	BankLast4         null.String `json:"bank_last4"`          // Last 4 digits of bank/card
	BankName          null.String `json:"bank_name"`           // Optional: bank name
	IsVerified        bool        `json:"is_verified"`         // Flag if bank account is verified
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type CreateACHChargeInput struct {
	CustomerID      string `json:"customer_id"`       // Required: Stripe Customer ID
	Amount          int64  `json:"amount"`            // Required: Amount in cents (e.g., 500 = $5.00)
	Currency        string `json:"currency"`          // e.g., "usd"
	PaymentMethodID string `json:"payment_method_id"` // Optional: Use a specific payment method ID
	IdempotencyKey  string `json:"idempotency_key"`   // Required: Prevents duplicate charges
	Description     string `json:"description"`       // Optional: Charge description
}

type ACHCharge struct {
	ID        string `json:"id"`         // Stripe charge ID
	Amount    int64  `json:"amount"`     // Charged amount in cents
	Currency  string `json:"currency"`   // e.g., "usd"
	Status    string `json:"status"`     // e.g., "succeeded", "pending", "failed"
	CreatedAt int64  `json:"created_at"` // Unix timestamp of the charge creation
}

type PaymentMethod struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`              // e.g., "us_bank_account", "card"
	CustomerID      string    `json:"customer_id"`       // Stripe customer ID
	Last4           string    `json:"last4"`             // Last 4 digits of bank/card
	BankName        string    `json:"bank_name"`         // Optional bank name
	IsDefault       bool      `json:"is_default"`        // Indicates if this is the default payment method
	CreatedAt       time.Time `json:"created_at"`
}

type Payment struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"`
	PlaidAccountID   string    `json:"plaid_account_id"`
	PlaidItemID      string    `json:"plaid_item_id"`
	StripeCustomerID string    `json:"stripe_customer_id"`
	StripePaymentID  string    `json:"stripe_payment_id"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
