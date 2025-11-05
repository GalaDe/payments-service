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

// PlaidWebhook godoc
// @Summary      Plaid webhook receiver
// @Description  Receives Plaid webhook events (e.g., TRANSACTIONS_UPDATED) and triggers internal updates
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        payload  body   object  true  "Plaid webhook payload"
// @Success      200
// @Failure      400  {string}  string  "Invalid Plaid webhook payload"
// @Router       /webhook/plaid [post]
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

// StripeWebhook godoc
// @Summary      Stripe webhook receiver
// @Description  Receives Stripe webhook events (e.g., charge.succeeded/failed) and updates payment status
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        Stripe-Signature  header  string  false  "Stripe signature header for webhook verification"
// @Param        payload           body    object  true   "Stripe event payload"
// @Success      200
// @Failure      400  {string}  string  "Invalid Stripe webhook payload"
// @Failure      503  {string}  string  "Error reading webhook request"
// @Router       /webhook/stripe [post]
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
