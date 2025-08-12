package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GalaDe/payments-service/internal/domain"
	"github.com/GalaDe/payments-service/internal/services/plaid"
	"github.com/gorilla/mux"
)

type CreateProcessorTokenRequest struct {
	AccessToken string `json:"accessToken"`
	AccountID   string `json:"accountID"`
}

type ExchangeTokenRequest struct {
	PublicToken string `json:"public_token"`
	UserID      string `json:"user_id"`
}

type ExchangeTokenResponse struct {
	AccessToken string `json:"access_token"`
	ItemID      string `json:"item_id"`
	AccountID   string `json:"account_id"`
}

/*

			Endpoint            | 				Description
| ----------------------------- | ------------------------------------------- |
| `POST /plaid/link-token`      | Create a link token for the frontend        |
| `POST /plaid/exchange`        | Exchange a public token for an access token |
| `GET  /plaid/accounts`        | Fetch linked bank accounts                  |
| `POST /plaid/processor-token` | Create a processor token for Stripe         |
| `DELETE /plaid/account/{id}`  | Unlink/delete a bank account                |

*/


/*

	POST /plaid/link-token

	1. Accept a user_id (or derive from auth/session)
	2. Call CreateLinkToken from your PlaidService
	3. Return a link_token in the response

	Link token is a short live(30 min), single use per session.
*/
func (h *HttpServer) CreateLinkToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	token, err := h.plaidService.CreateLinkToken(r.Context(), req.UserID)
	if err != nil {
		http.Error(w, "Failed to create link token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"link_token": token})
}

/*

	POST /plaid/exchange

	1. User opens Plaid Link using the link_token
	2. User selects and connects their bank account
	3. Plaid Link returns a public_token to the frontend
	4. Frontend sends that public_token to backend via POST /plaid/exchange

	Backend exchanges it for:

	1. access_token (used in API calls)
	2. item_id (Plaid's internal reference)
	3. account_id (the selected account)

	ExchangePublicToken exchanges a short-lived public_token for a long lived access_token
*/

func (h *HttpServer) ExchangePublicToken(w http.ResponseWriter, r *http.Request) {
	var req ExchangeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PublicToken == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.plaidService.ExchangePublicToken(r.Context(), req.PublicToken)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	plaidToken := domain.PlaidToken{
		UserID:      req.UserID,
		AccessToken: resp.AccessToken,
		AccountID:   resp.AccountID,
		ItemID:      resp.ItemID,
	}

	if err := h.repository.StorePlaidToken(r.Context(), plaidToken); err != nil {
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

/*
	GET  /plaid/accounts
*/
func (h *HttpServer) GetPlaidAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from query param or header
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	// Optional flag: ?with_balance=true
	withBal := r.URL.Query().Get("with_balance") == "true"

	token, err := h.repository.GetPlaidToken(ctx, userID)
	if err != nil {
		http.Error(w, "user has no linked account", http.StatusNotFound)
		return
	}

	if withBal {
		acc, err := h.plaidService.GetAccountWithBalance(ctx, token.AccessToken, token.AccountID)
		if err != nil {
			http.Error(w, "plaid balance error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode([]*plaid.AccountWithBalance{acc})
		return
	}

	acc, err := h.plaidService.GetAccount(ctx, token.AccessToken, token.AccountID)
	if err != nil {
		http.Error(w, "plaid account error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode([]*plaid.Account{acc})
}

/*
	POST /plaid/processor-token
*/
func (h *HttpServer) CreateProcessorTokenForStripe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, "Invalid request: missing user_id", http.StatusBadRequest)
		return
	}

	// Fetch stored Plaid token from DB
	plaidToken, err := h.repository.GetPlaidToken(ctx, req.UserID)
	if err != nil {
		http.Error(w, "Failed to get Plaid token: "+err.Error(), http.StatusNotFound)
		return
	}

	// Create Stripe bank account token using Plaid
	stripeToken, err := h.plaidService.CreateStripeToken(ctx, plaidToken.AccessToken, plaidToken.AccountID)
	if err != nil {
		http.Error(w, "Failed to create Stripe token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"stripe_bank_account_token": *stripeToken,
	})
}

/*
	DELETE /plaid/account/{id}

	Removes a user's linked Plaid account (and token) from both:

		- Plaid (via ItemRemove)
		- Database (via repository method like DeletePlaidToken)
*/

func (h *HttpServer) DeletePlaidAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := mux.Vars(r)["id"]

	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	if err := h.repository.DeletePlaidToken(ctx, userID); err != nil {
		http.Error(w, "Failed to delete Plaid token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
