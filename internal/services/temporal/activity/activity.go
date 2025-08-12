package activity

import (
	"context"
	"fmt"

	"github.com/GalaDe/payments-service/internal/domain"
	"github.com/GalaDe/payments-service/internal/services/plaid"
	"github.com/GalaDe/payments-service/internal/services/stripe"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type TemporalActivityPort struct {
	repository     domain.Repository
	stripe         stripe.StripeService
	plaid          plaid.PlaidService
	temporalClient client.Client
}

func NewTemporalActivityPort(repository domain.Repository, stripe stripe.StripeService, plaid plaid.PlaidService, temporalClient client.Client) *TemporalActivityPort {
	return &TemporalActivityPort{
		repository,
		stripe,
		plaid,
		temporalClient,
	}
}

const (
	EnsureDefaultPaymentMethodActivity = "EnsureDefaultPaymentMethodActivity"
	EnsurePlaidAccountActivity         = "EnsurePlaidAccountActivity"
	GetOrCreateStripeCustomerActivity  = "GetOrCreateStripeCustomerActivity"
	CreateACHCharge                    = "CreateACHCharge"
)

func (a *TemporalActivityPort) RegisterActivities(w worker.ActivityRegistry) {
	w.RegisterActivityWithOptions(a.ensureDefaultPaymentMethodActivity, activity.RegisterOptions{Name: EnsureDefaultPaymentMethodActivity})
	w.RegisterActivityWithOptions(a.ensurePlaidAccountActivity, activity.RegisterOptions{Name: EnsurePlaidAccountActivity})
	w.RegisterActivityWithOptions(a.getOrCreateStripeCustomerActivity, activity.RegisterOptions{Name: GetOrCreateStripeCustomerActivity})
	w.RegisterActivityWithOptions(a.stripe.CreateACHCharge, activity.RegisterOptions{Name: CreateACHCharge})
}

/*
	Before charging a customer, Stripe requires a default payment method. This activity guarantees that requirement is met.

	Use Case:
		1. A customer previously linked their bank via Plaid and you created a Stripe payment method from it.
		2. A customer exists in Stripe but hasn't linked a payment method yet.
		3. You want to ensure idempotency: reuse existing payment methods when possible.

*/

type EnsureDefaultPaymentMethodInput struct {
	CustomerID string
	UserID     string
}

func (a *TemporalActivityPort) ensureDefaultPaymentMethodActivity(ctx context.Context, input EnsureDefaultPaymentMethodInput) (*domain.PaymentMethod, error) {
	// Check DB for StripeCustomer
	customer, err := a.repository.GetStripeCustomerByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("no Stripe customer for user %s: %w", input.UserID, err)
	}

	// If already has default payment method, return it
	if customer.DefaultPaymentID.Valid {
		pm, err := a.stripe.RetrievePaymentMethod(ctx, customer.DefaultPaymentID.String)
		if err == nil {
			return pm, nil
		}
	}

	// Get Plaid processor token
	token, err := a.repository.GetPlaidToken(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("no Plaid token for user: %w", err)
	}

	// Use your reusable method
	pm, err := a.stripe.CreatePaymentMethodFromBankToken(ctx, customer.StripeCustomerID, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment method: %w", err)
	}

	// Set as default using your service method
	err = a.stripe.UpdateDefaultStripePaymentMethod(ctx, &stripe.UpdateDefaultStripePaymentMethodInput{
		CustomerID:      customer.StripeCustomerID,
		PaymentMethodID: pm.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set default: %w", err)
	}

	return pm, nil
}

/*
	Check if the user has a Plaid account linked in your database (e.g., plaid_tokens table):
		- If yes → return access token + account ID.
		- If no → return an error or trigger a setup flow (e.g., send link_token back to frontend).

*/

type EnsurePlaidAccountOutput struct {
	AccessToken string
	AccountID   string
}

func (a *TemporalActivityPort) ensurePlaidAccountActivity(ctx context.Context, userID string) (*EnsurePlaidAccountOutput, error) {
	// Step 1: Query your DB for plaid_tokens
	token, err := a.repository.GetPlaidToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("plaid account not linked for user %s: %w", userID, err)
	}

	// Step 2: Return accessToken and accountID
	return &EnsurePlaidAccountOutput{
		AccessToken: token.AccessToken,
		AccountID:   token.AccountID,
	}, nil
}

/*
	When you’re initiating payments, linking bank accounts, or creating payment methods, Stripe expects a Customer to exist first.
	This activity guarantees that requirement is met.
*/

type GetOrCreateStripeCustomerInput struct {
	UserID string
	Email  string
}

func (a *TemporalActivityPort) getOrCreateStripeCustomerActivity(ctx context.Context, input GetOrCreateStripeCustomerInput) (*domain.StripeCustomer, error) {
	// 1. Check if customer exists in DB
	cust, err := a.repository.GetStripeCustomerByUserID(ctx, input.UserID)
	if err == nil {
		// Already exists
		return &domain.StripeCustomer{
			StripeCustomerID: cust.StripeCustomerID,
			Email:            cust.Email,
		}, nil
	}

	// 2. Doesn't exist → Create in Stripe
	newCustomer, err := a.stripe.CreateStripeCustomer(&stripe.CreateStripeCustomerInput{
		UserID: &input.UserID,
		Email:  &input.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("stripe customer creation failed: %w", err)
	}

	// 3. Save in DB
	err = a.repository.InsertStripeCustomer(ctx, &domain.StripeCustomer{
		UserID:           input.UserID,
		StripeCustomerID: newCustomer.StripeCustomerID,
		Email:            newCustomer.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store stripe customer: %w", err)
	}

	return newCustomer, nil
}
