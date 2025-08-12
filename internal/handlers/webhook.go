package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v75"
)

/*

| Endpoint               | Description                                            |
| ---------------------- | ------------------------------------------------------ |
| `POST /webhook/plaid`  | Receive events from Plaid (e.g., transactions updated) |
| `POST /webhook/stripe` | Handle Stripe events (payment succeeded, failed, etc.) |

*/

func (h *HttpServer) PlaidWebhook(w http.ResponseWriter, r *http.Request) {
	var webhookEvent map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&webhookEvent); err != nil {
		http.Error(w, "Invalid Plaid webhook payload", http.StatusBadRequest)
		return
	}

	// Example: Handle TRANSACTIONS_UPDATED
	if webhookEvent["webhook_type"] == "TRANSACTIONS" && webhookEvent["webhook_code"] == "TRANSACTIONS_UPDATED" {
		log.Printf("Plaid transactions updated: %+v", webhookEvent)
		// Optionally: update local transaction cache, trigger downstream workflows, etc.
	}

	w.WriteHeader(http.StatusOK)
}

func (h *HttpServer) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading webhook request", http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}
	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "Invalid Stripe webhook payload", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "charge.succeeded":
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err == nil {
			log.Printf("Charge succeeded for: %s", charge.ID)
			// Update payment status in DB
		}
	case "charge.failed":
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err == nil {
			log.Printf("Charge failed for: %s", charge.ID)
			// Update payment status in DB
		}
	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}
