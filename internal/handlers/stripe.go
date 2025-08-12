package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

/*
| Endpoint                             | Description                                         |
| ------------------------------------ | --------------------------------------------------- |
| `POST /stripe/payment-method`        | Create a payment method using Plaid processor token |
| `GET  /stripe/payment-methods`       | List stored payment methods                         |
| `DELETE /stripe/payment-method/{id}` | Delete a payment method                             |
*/

type CreatePaymentMethodRequest struct {
	ProcessorToken string `json:"processor_token"`
	CustomerID     string `json:"customer_id"`
}

/*
	POST /stripe/payment-method
*/

func (h *HttpServer) CreateStripePaymentMethod(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProcessorToken == "" || req.CustomerID == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call the service to create the payment method
	pm, err := h.stripeService.CreatePaymentMethodFromBankToken(r.Context(), req.CustomerID, req.ProcessorToken)
	if err != nil {
		http.Error(w, "Stripe payment method creation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(pm)
}

/*
	GET  /stripe/payment-methods
*/

func (h *HttpServer) GetStripePaymentMethod(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		http.Error(w, "Missing required query parameter: customer_id", http.StatusBadRequest)
		return
	}

	// Optional: filter by type (e.g., "us_bank_account")
	paymentMethods, err := h.stripeService.GetCustomerPaymentMethods(r.Context(), customerID, "us_bank_account")
	if err != nil {
		http.Error(w, "Failed to retrieve payment methods: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paymentMethods)

}

/*
	DELETE /stripe/payment-method/{id}
*/

func (h *HttpServer) DeleteStripePaymentMethod(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	paymentMethodID := mux.Vars(r)["payment_method_id"]

	if paymentMethodID == "" {
		http.Error(w, "Missing payment_method_id", http.StatusBadRequest)
		return
	}

	err := h.stripeService.DeleteStripePaymentMethod(ctx, paymentMethodID)
	if err != nil {
		http.Error(w, "Failed to delete Stripe payment method: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
