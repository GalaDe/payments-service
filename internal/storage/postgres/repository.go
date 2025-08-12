
package postgres

import (
	"context"
	"fmt"

	"github.com/GalaDe/payments-service/internal/domain"
	orm "github.com/GalaDe/payments-service/internal/sqlc"
	"github.com/GalaDe/payments-service/internal/utils"
	"github.com/google/uuid"
)

type postgresRepo struct {
	tx *PostgresTransactor
}

func NewPostgresRepo(tx *PostgresTransactor) domain.Repository {
	return &postgresRepo{tx}
}

func (p *postgresRepo) StorePlaidToken(ctx context.Context, token domain.PlaidToken) error {
	q := p.tx.WithQtx(ctx)
	return q.UpsertPlaidToken(ctx, orm.UpsertPlaidTokenParams{
		UserID:      token.UserID,
		AccessToken: token.AccessToken,
		AccountID:   token.AccountID,
		ItemID:      token.ItemID,
	})
}

func (p *postgresRepo) GetPlaidToken(ctx context.Context, userID string) (*domain.PlaidToken, error) {
	q := p.tx.WithQtx(ctx)
	dbToken, err := q.GetPlaidTokenByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &domain.PlaidToken{
		UserID:      userID,
		AccessToken: dbToken.AccessToken,
		AccountID:   dbToken.AccountID,
		ItemID:      dbToken.ItemID,
	}, nil
}

func (p *postgresRepo) DeletePlaidToken(ctx context.Context, userID string) error {
	q := p.tx.WithQtx(ctx)
	return q.DeletePlaidToken(ctx, userID)
}

// TODO: Revise SqlToNullString conversion, I belive it can be done better
func (r *postgresRepo) GetStripeCustomerByUserID(ctx context.Context, userID string) (*domain.StripeCustomer, error) {
	q := r.tx.WithQtx(ctx)

	dbCust, err := q.GetStripeCustomerByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe customer for user %s: %w", userID, err)
	}

	return &domain.StripeCustomer{
		UserID:            dbCust.UserID,
		StripeCustomerID:  dbCust.StripeCustomerID,
		Email:             utils.SqlToNullString(dbCust.Email),
		DefaultPaymentID:  utils.SqlToNullString(dbCust.DefaultPaymentID),
		PaymentMethodType: utils.SqlToNullString(dbCust.PaymentMethodType),
		BankLast4:         utils.SqlToNullString(dbCust.BankLast4),
		BankName:          utils.SqlToNullString(dbCust.BankName),
		IsVerified:        dbCust.IsVerified.Valid && dbCust.IsVerified.Bool,
		CreatedAt:         dbCust.CreatedAt.Time,
		UpdatedAt:         dbCust.UpdatedAt.Time,
	}, nil
}

// TODO: Revise NullStringToSQL conversion, I belive it can be done better
func (r *postgresRepo) InsertStripeCustomer(ctx context.Context, customer *domain.StripeCustomer) error {
	q := r.tx.WithQtx(ctx)

	return q.InsertStripeCustomer(ctx, orm.InsertStripeCustomerParams{
		UserID:            customer.UserID,
		StripeCustomerID:  customer.StripeCustomerID,
		Email:             utils.NullStringToSQL(customer.Email),
		DefaultPaymentID:  utils.NullStringToSQL(customer.DefaultPaymentID),
		PaymentMethodType: utils.NullStringToSQL(customer.PaymentMethodType),
		BankLast4:         utils.NullStringToSQL(customer.BankLast4),
		BankName:          utils.NullStringToSQL(customer.BankName),
		IsVerified:        utils.NullBoolToSQL(customer.IsVerified),
	})
}

func (r *postgresRepo) InsertPayment(ctx context.Context, payment *domain.Payment) error {
	q := r.tx.WithQtx(ctx)

	return q.InsertPayment(ctx, orm.InsertPaymentParams{
		UserID:           payment.UserID,
		Amount:           payment.Amount,
		Currency:         payment.Currency,
		PlaidAccountID:   utils.StringToNull(payment.PlaidAccountID),
		PlaidItemID:      utils.StringToNull(payment.PlaidItemID),
		StripeCustomerID: utils.StringToNull(payment.StripeCustomerID),
		StripePaymentID:  utils.StringToNull(payment.StripePaymentID),
		Status:           payment.Status,
	})
}

func (r *postgresRepo) UpdatePaymentStatus(ctx context.Context, paymentID, status string) error {
	id, err := uuid.Parse(paymentID)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}

	q := r.tx.WithQtx(ctx)
	return q.UpdatePaymentStatus(ctx, orm.UpdatePaymentStatusParams{
		ID:     id,
		Status: status,
	})
}

func (r *postgresRepo) GetPaymentByID(ctx context.Context, paymentID string) (*domain.Payment, error) {
	q := r.tx.WithQtx(ctx)

	id, err := uuid.Parse(paymentID)
	if err != nil {
		return nil, err
	}
	dbPayment, err := q.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.Payment{
		ID:               dbPayment.ID.String(),
		UserID:           dbPayment.UserID,
		Amount:           dbPayment.Amount,
		Currency:         dbPayment.Currency,
		PlaidAccountID:   dbPayment.PlaidAccountID.String,
		PlaidItemID:      dbPayment.PlaidItemID.String,
		StripeCustomerID: dbPayment.StripeCustomerID.String,
		StripePaymentID:  dbPayment.StripePaymentID.String,
		Status:           dbPayment.Status,
		CreatedAt:        dbPayment.CreatedAt.Time,
		UpdatedAt:        dbPayment.UpdatedAt.Time,
	}, nil
}

func (r *postgresRepo) GetAllPayments(ctx context.Context) ([]*domain.Payment, error) {
	q := r.tx.WithQtx(ctx)

	dbPayments, err := q.GetAllPayments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get payments: %w", err)
	}

	payments := make([]*domain.Payment, 0, len(dbPayments))
	for _, p := range dbPayments {
		payments = append(payments, &domain.Payment{
			ID:               p.ID.String(),
			UserID:           p.UserID,
			Amount:           p.Amount,
			Currency:         p.Currency,
			PlaidAccountID:   utils.NullStringToStr(p.PlaidAccountID),
			PlaidItemID:      utils.NullStringToStr(p.PlaidItemID),
			StripeCustomerID: utils.NullStringToStr(p.StripeCustomerID),
			StripePaymentID:  utils.NullStringToStr(p.StripePaymentID),
			Status:           p.Status,
			CreatedAt:        p.CreatedAt.Time,
			UpdatedAt:        p.UpdatedAt.Time,
		})
	}

	return payments, nil
}
