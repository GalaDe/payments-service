package domain

import "context"


type Repository interface {
	GetPlaidToken(ctx context.Context, userID string) (*PlaidToken, error)
	StorePlaidToken(ctx context.Context, token PlaidToken) error
	DeletePlaidToken(ctx context.Context, userID string) error
	GetStripeCustomerByUserID(ctx context.Context, userID string)(*StripeCustomer, error) 
	InsertStripeCustomer(ctx context.Context, customer *StripeCustomer) error 
	InsertPayment(ctx context.Context, payment *Payment) error 
	UpdatePaymentStatus(ctx context.Context, paymentID, status string) error
	GetPaymentByID(ctx context.Context, paymentID string) (*Payment, error)
	GetAllPayments(ctx context.Context) ([]*Payment, error)
}
