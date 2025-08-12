package workflow

import (
	"go.temporal.io/sdk/workflow"

	"github.com/GalaDe/payments-service/internal/domain"
	activity "github.com/GalaDe/payments-service/internal/services/temporal/activity"
)

type PaymentWorkflowInput struct {
	UserID           string `json:"user_id"`           // internal app user
	CustomerID       string `json:"customer_id"`       // Stripe customer ID
	PaymentMethodID  string `json:"payment_method_id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Description      string `json:"description"`
	IdempotencyKey   string `json:"idempotency_key"`
}

/*
 1. Check if Plaid/Stripe account setup exists
 2. If not:
    a. Retrieve Plaid token from DB (or error out)
    b. Create Stripe customer (if needed)
    c. Create Stripe bank account or payment method (if needed)
 3. Proceed to charge the customer (ACH)
 4. Update DB
*/
func paymentWorkflow(ctx workflow.Context, input PaymentWorkflowInput) error {
	// Set retry policy or activity timeout if needed
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: DefaultActivityTimeout,
		RetryPolicy:         RetryPolicy3Attempts,
	})

	// Step 1: Ensure user has Plaid token
	var plaidToken *domain.PlaidToken
	if err := workflow.ExecuteActivity(ctx, activity.EnsurePlaidAccountActivity, input.UserID).Get(ctx, &plaidToken); err != nil {
		return err
	}

	// Step 2: Get or create Stripe customer
	var stripeCustomer *domain.StripeCustomer
	if err := workflow.ExecuteActivity(ctx, activity.GetOrCreateStripeCustomerActivity, input.UserID).Get(ctx, &stripeCustomer); err != nil {
		return err
	}

	// Step 3: Ensure default payment method exists
	if err := workflow.ExecuteActivity(ctx, activity.EnsureDefaultPaymentMethodActivity, stripeCustomer).Get(ctx, nil); err != nil {
		return err
	}

	// Step 4: Charge customer
	chargeInput := domain.CreateACHChargeInput{
		CustomerID:     stripeCustomer.StripeCustomerID,
		Amount:         input.Amount,
		IdempotencyKey: input.IdempotencyKey,
	}
	var charge *domain.ACHCharge
	if err := workflow.ExecuteActivity(ctx, activity.CreateACHCharge, chargeInput).Get(ctx, &charge); err != nil {
		return err
	}

	// Step 5: Save payment record (optional)
	// ... add SavePaymentRecordActivity if needed

	return nil
}
