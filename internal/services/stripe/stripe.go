package stripe

import (
	"context"
	"fmt"

	"github.com/GalaDe/payments-service/internal/domain"
	"github.com/google/uuid"
	"github.com/guregu/null"
	"github.com/stripe/stripe-go/v75"

	//"github.com/stripe/stripe-go/v75/bankaccount"
	"github.com/stripe/stripe-go/v75/charge"
	"github.com/stripe/stripe-go/v75/customer"
	"github.com/stripe/stripe-go/v75/paymentmethod"
	"github.com/stripe/stripe-go/v75/token"
)

type stripeImpl struct {
	Config *StripeConfig
}

type StripeConfig struct {
	AppKey      string `json:"AppKey"`
	WebhookKey  string `json:"WebhookKey"`
	Environment string `json:"Environment"`
}

type StripeService interface {
	CreateStripeCustomer(input *CreateStripeCustomerInput) (*domain.StripeCustomer, error)
	// CreateStripeBankAccount(cust *stripe.Customer, bankAccountToken *string) (*stripe.BankAccount, error)
	CreatePaymentMethodFromBankToken(ctx context.Context, customerID string, processorToken string) (*domain.PaymentMethod, error)
	GetCustomerPaymentMethods(ctx context.Context, customerID string, paymentType string) ([]*stripe.PaymentMethod, error)
	UpdateDefaultStripePaymentMethod(ctx context.Context, input *UpdateDefaultStripePaymentMethodInput) error
	//DeleteStripeBankAccount(input *DeleteStripeBankAccountInput) (*stripe.BankAccount, error)
	DeleteStripePaymentMethod(ctx context.Context, paymentMethodID string) error
	//UpdateDefaultStripeBankAccount(ctx context.Context, input *UpdateDefaultStripeBankAccountInput) error
	CreateACHCharge(ctx context.Context, input *CreateACHChargeInput) (*ACHCharge, error)
	RetrieveStripeToken(ctx context.Context, tokenID string) (*stripe.Token, error)
	RetrievePaymentMethod(ctx context.Context, paymentMethodID string) (*domain.PaymentMethod, error)
}

func NewStripe(config *StripeConfig) StripeService {
	return &stripeImpl{Config: config}
}

type ACHCharge struct {
	ID     string
	Amount int
	Status string
}

type CreateStripeCustomerInput struct {
	UserID *string `json:"user_id"`
	Email  *string `json:"email"`
}

type UpdateDefaultStripePaymentMethodInput struct {
	CustomerID      string `json:"customer_id"`
	PaymentMethodID string `json:"payment_method_id"`
}

// type UpdateDefaultStripeBankAccountInput struct {
// 	CustomerID    string `json:"CustomerID"`
// 	BankAccountID string `json:"BankAccountID"`
// 	IsDefault     bool   `json:"IsDefault"`
// }

// type DeleteStripeBankAccountInput struct {
// 	CustomerID       string `json:"CustomerID"`
// 	BankAccountID    string `json:"BankAccountID"`
// 	BankAccountToken string `json:"BankAccountToken"`
// }

type CreateACHChargeInput struct {
	CustomerID     string `json:"CustomerID"`
	Amount         int64  `json:"Amount"`
	IdempotencyKey string `json:"IdempotencyKey"`
}

// createStripeCustomer function represents the user in the Stripe system.
func (s *stripeImpl) CreateStripeCustomer(input *CreateStripeCustomerInput) (*domain.StripeCustomer, error) {
	stripe.Key = s.Config.AppKey

	params := &stripe.CustomerParams{}

	if input.Email != nil {
		params.Email = stripe.String(*input.Email)
	}

	if input.UserID != nil {
		params.Metadata = map[string]string{
			"user_id": *input.UserID,
		}
	}

	stripeCustomer, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return &domain.StripeCustomer{
		UserID:           *input.UserID,
		StripeCustomerID: stripeCustomer.ID,
		Email:            null.NewString(stripeCustomer.Email, stripeCustomer.Email != ""),
	}, nil
}

/*
Attach the bank account information obtained from Plaid (as a bank account token)
to the Stripe customer. This is a necessary step because it links the user's bank account
information to their Stripe customer profile, enabling to use it for future transactions.
*/
// func (s *stripeImpl) CreateStripeBankAccount(cust *stripe.Customer, bankAccountToken *string) (*stripe.BankAccount, error) {
// 	stripe.Key = s.Config.AppKey

// 	bankAccountParams := &stripe.BankAccountParams{
// 		Token:    bankAccountToken,
// 		Customer: &cust.ID,
// 	}

// 	bankAccount, err := bankaccount.New(bankAccountParams)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return bankAccount, nil
// }

// CreatePaymentMethodFromBankToken creates a PaymentMethod from a Plaid processor_token (btok_...)
func (s *stripeImpl) CreatePaymentMethodFromBankToken(ctx context.Context, customerID string, processorToken string) (*domain.PaymentMethod, error) {
	params := &stripe.PaymentMethodParams{
		Customer:      stripe.String(customerID),
		Type:          stripe.String("us_bank_account"),
		USBankAccount: &stripe.PaymentMethodUSBankAccountParams{
			// 🔥 Stripe DOES NOT allow this anymore (no Token field here)
			// So we skip this and instead use this format:
		},
		// This is how you attach the Plaid token:
		PaymentMethod: stripe.String(processorToken), // this is "btok_..."
	}

	pm, err := paymentmethod.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: failed to create payment method: %w", err)
	}

	return &domain.PaymentMethod{
		ID:         pm.ID,
		CustomerID: customerID,
		Type:       string(pm.Type),
		BankName:   pm.USBankAccount.BankName,
		Last4:      pm.USBankAccount.Last4,
		IsDefault:  false,
	}, nil
}

func (s *stripeImpl) GetCustomerPaymentMethods(ctx context.Context, customerID string, paymentType string) ([]*stripe.PaymentMethod, error) {
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String(paymentType), // e.g., "us_bank_account"
	}
	params.Context = ctx

	iter := paymentmethod.List(params)

	var result []*stripe.PaymentMethod
	for iter.Next() {
		result = append(result, iter.PaymentMethod())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list Stripe payment methods: %w", err)
	}

	return result, nil
}

// func (s *stripeImpl) UpdateDefaultStripeBankAccount(ctx context.Context, input *UpdateDefaultStripeBankAccountInput) error {
// 	stripe.Key = s.Config.AppKey

// 	customerParams := &stripe.CustomerParams{
// 		DefaultSource: stripe.String(input.BankAccountID),
// 		Params: stripe.Params{
// 			Context: ctx,
// 		},
// 	}
// 	_, err := customer.Update(input.CustomerID, customerParams)
// 	if err != nil {
// 		return err
// 	}

// 	return err
// }

func (s *stripeImpl) UpdateDefaultStripePaymentMethod(ctx context.Context, input *UpdateDefaultStripePaymentMethodInput) error {
	stripe.Key = s.Config.AppKey

	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(input.PaymentMethodID),
		},
	}
	customerParams.Context = ctx

	_, err := customer.Update(input.CustomerID, customerParams)
	if err != nil {
		return fmt.Errorf("failed to set default payment method for customer %s: %w", input.CustomerID, err)
	}

	return nil
}

// func (s *stripeImpl) DeleteStripeBankAccount(input *DeleteStripeBankAccountInput) (*stripe.BankAccount, error) {
// 	stripe.Key = s.Config.AppKey

// 	bankAccountParams := &stripe.BankAccountParams{
// 		Customer: &input.CustomerID,
// 	}
// 	bankAccountParams.SetIdempotencyKey(fmt.Sprintf("%s-%s", input.CustomerID, input.BankAccountID))

// 	bankAccount, err := bankaccount.Del(input.BankAccountID, bankAccountParams)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return bankAccount, nil
// }

func (s *stripeImpl) DeleteStripePaymentMethod(ctx context.Context, paymentMethodID string) error {
	stripe.Key = s.Config.AppKey

	_, err := paymentmethod.Detach(paymentMethodID, nil)
	if err != nil {
		return fmt.Errorf("failed to detach payment method: %w", err)
	}
	return nil
}

func (s *stripeImpl) CreateACHCharge(ctx context.Context, input *CreateACHChargeInput) (*ACHCharge, error) {
	stripe.Key = s.Config.AppKey

	chargeParams := &stripe.ChargeParams{
		Amount:      &input.Amount, // amount in cents
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Customer:    stripe.String(input.CustomerID),
		Description: stripe.String("ACH charge for user " + input.CustomerID),
		Params: stripe.Params{
			IdempotencyKey: stripe.String(input.IdempotencyKey),
			Context:        ctx,
		},
	}

	result, err := charge.New(chargeParams)
	if err != nil {
		return nil, fmt.Errorf("stripe charge error: %w", err)
	}

	return &ACHCharge{
		ID:     result.ID,
		Amount: int(result.Amount),
		Status: string(result.Status),
	}, nil
}

// This function will return the underlying information that is associated with a stripe token
// In our case this is usually a bank token.
// IMPORTANT NOTE: Due to the limitations of how Plaid + Stripe interact with each other in
// the sandbox environment, we will randomly generate our own bank account fingerprint.
// This is because Plaid + Stripe uses the same bank account no matter what selection is made
// in Plaid link. As a result, this will cause the stripe bank account fingerprint to always be
// the same value resulting in errors. Therefore, we will always create our own bank account
// fingerprint when using the Stripe sandbox credentials. PLEASE NOTE that while creating
// a customer fingerprint in the sandbox environment may bypass issues in our system, Stripe
// will still do its own validations which may break any downstream process.
// Meaning while we can create multiple bank accounts on our end, Stripe will still not.
// [Retrieve a Token] https://docs.stripe.com/api/tokens/retrieve
func (s *stripeImpl) RetrieveStripeToken(ctx context.Context, tokenID string) (*stripe.Token, error) {
	stripe.Key = s.Config.AppKey

	token, err := token.Get(tokenID, &stripe.TokenParams{})
	if err != nil {
		return nil, err
	}

	// This code is only executed when using the sandbox environment
	if s.Config.Environment == "sandbox" && token.BankAccount != nil {
		token.BankAccount.Fingerprint = uuid.NewString()
	}

	return token, nil
}

/*
The RetrievePaymentMethod function is used to fetch details about a saved payment method from Stripe using its ID.

Use Case:
 1. Get full metadata about the stored payment method (like bank name, last 4 digits, status, etc.).
 2. Verify that the method is still valid (not detached, expired, or disabled).
 3. Display payment details to the user (e.g., in a dashboard or checkout summary).
 4. Check if the default payment method is usable before charging.
*/
func (s *stripeImpl) RetrievePaymentMethod(ctx context.Context, paymentMethodID string) (*domain.PaymentMethod, error) {
	stripe.Key = s.Config.AppKey

	pm, err := paymentmethod.Get(paymentMethodID, &stripe.PaymentMethodParams{
		Params: stripe.Params{
			Context: ctx,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment method: %w", err)
	}
	return &domain.PaymentMethod{
		ID:         pm.ID,
		CustomerID: stripe.StringValue(&pm.Customer.ID),
		Type:       string(pm.Type),
		BankName:   pm.USBankAccount.BankName,
		Last4:      pm.USBankAccount.Last4,
		IsDefault:  false,
	}, nil
}
