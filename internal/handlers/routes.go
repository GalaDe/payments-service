package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(h *HttpServer) *chi.Mux {
	r := chi.NewRouter()

	// Health check or default route
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Payments service is running"))
	})

	// Plaid routes
	r.Post("/plaid/link-token", h.CreateLinkToken)
	r.Post("/plaid/exchange", h.ExchangePublicToken)
	r.Get("/plaid/accounts", h.GetPlaidAccounts)
	r.Post("/plaid/processor-token", h.CreateProcessorTokenForStripe)
	r.Delete("/plaid/account/{id}", h.DeletePlaidAccount)

	// Stripe routes
	r.Post("/stripe/payment-method", h.CreateStripePaymentMethod)
	r.Get("/stripe/payment-methods", h.GetStripePaymentMethod)
	r.Delete("/stripe/payment-method/{id}", h.DeleteStripePaymentMethod)

	// Payment routes
	r.Post("/payments", h.CreatePayment)
	r.Get("/payments", h.GetPayments)
	r.Get("/payments/{id}", h.GetPaymentByID)

	// Webhooks
	r.Post("/webhook/plaid", h.PlaidWebhook)
	r.Post("/webhook/stripe", h.StripeWebhook)

	return r
}
