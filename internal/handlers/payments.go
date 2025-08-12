package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/GalaDe/payments-service/internal/services/temporal/workflow"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.temporal.io/sdk/client"
)

/*


Payments APIs (Temporal-powered)


| Endpoint              | Description                                     |
| --------------------- | ----------------------------------------------- |
| `POST /payments`      | Initiate a payment (starts a Temporal workflow) |
| `GET  /payments/{id}` | Check payment status                            |
| `GET  /payments`      | List user’s payments (optionally with filters)  |


*/

type CreatePaymentRequest struct {
	UserID          string `json:"user_id"`
	CustomerID      string `json:"customer_id"`
	PaymentMethodID string `json:"payment_method_id"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	Description     string `json:"description"`
}

/*
POST /payments
*/
func (h *HttpServer) CreatePayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreatePaymentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.CustomerID == "" || req.PaymentMethodID == "" || req.Amount <= 0 || req.Currency == "" {
		http.Error(w, "Missing or invalid fields", http.StatusBadRequest)
		return
	}

	workflowID := fmt.Sprintf("payment-%s-%s", req.CustomerID, uuid.NewString())

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "payment-task-queue",
	}

	workflowInput := workflow.PaymentWorkflowInput{
		UserID:          req.UserID,
		CustomerID:      req.CustomerID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Description:     req.Description,
		IdempotencyKey:  fmt.Sprintf("%s-%d", req.CustomerID, time.Now().UnixNano()), // Example key
	}

	we, err := h.worker.ExecuteWorkflow(ctx, workflowOptions, workflow.PaymentWorkflow, workflowInput)
	if err != nil {
		log.Printf("Failed to start payment workflow: %v", err)
		http.Error(w, "Failed to start payment workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Started payment workflow: workflow_id=%s run_id=%s", we.GetID(), we.GetRunID())

	json.NewEncoder(w).Encode(map[string]string{
		"workflow_id": we.GetID(),
		"run_id":      we.GetRunID(),
		"status":      "started",
	})
}

/*
	GET  /payments/{id}
*/

func (h *HttpServer) GetPaymentByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	paymentID := mux.Vars(r)["id"]

	if paymentID == "" {
		http.Error(w, "Missing payment ID", http.StatusBadRequest)
		return
	}

	payment, err := h.repository.GetPaymentByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Payment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(payment)
}

/*
	GET  /payments
*/
func (h *HttpServer) GetPayments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	payments, err := h.repository.GetAllPayments(ctx)
	if err != nil {
		http.Error(w, "Failed to retrieve payments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payments); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

