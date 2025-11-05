package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GalaDe/payments-service/internal/domain"
	"github.com/GalaDe/payments-service/internal/services/plaid"
)

type CreateProcessorTokenRequest struct {
	AccessToken string `json:"accessToken"`
	AccountID   string `json:"accountID"`
	UserID      string `json:"user_id" example:"user_123"`
}

// CreateLinkTokenRequest is the body for POST /plaid/link-token
type CreateLinkTokenRequest struct {
	UserID string `json:"user_id" example:"user_123"`
}

type GetPlaidAccountsRequest struct {
	UserID      string `json:"user_id"`
	WithBalance bool   `json:"with_balance"`
}

type ExchangeTokenRequest struct {
	PublicToken string `json:"public_token"`
	UserID      string `json:"user_id"`
}

type DeletePlaidAccountRequest struct{
	UserID      string `json:"id"`
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

// CreateLinkToken godoc
// @Summary      Create Plaid Link Token
// @Description  Generates a new Plaid Link Token for the given user ID
// @Tags         plaid
// @Accept       json
// @Produce      json
// @Param        request  body  CreateLinkTokenRequest  true  "User ID request"
// @Success      200  {object}  map[string]string
// @Failure      400  {string}  string "Invalid request"
// @Failure      500  {string}  string "Failed to create link token"
// @Router       /plaid/link-token [post]
func (h *HttpServer) CreateLinkToken(w http.ResponseWriter, r *http.Request) {

	var req CreateLinkTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, "Invalid request: missing user_id", http.StatusBadRequest)
		return
	}

	token, err := h.plaidService.CreateLinkToken(r.Context(), req.UserID)
	if err != nil {
		http.Error(w, "Failed to create link token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"link_token": token})
}

// ExchangePublicToken godoc
// @Summary      Exchange Plaid Public Token
// @Description  Exchanges a Plaid public token for an access token, stores it, and links it to the user
// @Tags         plaid
// @Accept       json
// @Produce      json
// @Param        request  body  ExchangeTokenRequest  true  "Public token exchange request"
// @Success      200  {object}  map[string]string
// @Failure      400  {string}  string "Invalid request"
// @Failure      500  {string}  string "Failed to exchange or save token"
// @Router       /plaid/exchange-token [post]
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

// GetPlaidAccounts godoc
// @Summary      Get user's linked Plaid account
// @Description  Returns the user's linked Plaid account. Use `with_balance=true` to include current/available balance.
// @Tags         plaid
// @Accept       json
// @Produce      json
// @Param        user_id       query   string  true  "User ID whose Plaid account to fetch"
// @Param        with_balance  query   bool    false "If true, returns account with balance"
// @Success      200  {array}  plaid.Account
// @Success      200  {array}  plaid.AccountWithBalance
// @Failure      400  {string} string "missing user_id"
// @Failure      404  {string} string "user has no linked account"
// @Failure      500  {string} string "plaid error"
// @Router       /plaid/accounts [get]
func (h *HttpServer) GetPlaidAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req GetPlaidAccountsRequest

	// Optional flag: ?with_balance=true
	token, err := h.repository.GetPlaidToken(ctx, req.UserID)
	if err != nil {
		http.Error(w, "user has no linked account", http.StatusNotFound)
		return
	}

	if req.WithBalance {
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

// CreateProcessorTokenForStripe godoc
// @Summary      Create Stripe bank-account token from Plaid data
// @Description  Looks up the user's stored Plaid access/account IDs and creates a Stripe processor bank token (btok_…)
// @Tags         plaid
// @Accept       json
// @Produce      json
// @Param        request  body  CreateProcessorTokenRequest  true  "User ID payload"
// @Success      200  {object}  map[string]string  "keys: stripe_bank_account_token"
// @Failure      400  {string}  string "Invalid request: missing user_id"
// @Failure      404  {string}  string "Failed to get Plaid token"
// @Failure      500  {string}  string "Failed to create Stripe token"
// @Router       /plaid/processor-token [post]
func (h *HttpServer) CreateProcessorTokenForStripe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateProcessorTokenRequest
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

// DeletePlaidAccount godoc
// @Summary      Delete Plaid account
// @Description  Removes a stored Plaid token for a given user ID
// @Tags         plaid
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      204  "No Content"
// @Failure      400  {string}  string "Missing user ID"
// @Failure      500  {string}  string "Failed to delete Plaid token"
// @Router       /plaid/account/{id} [delete]
func (h *HttpServer) DeletePlaidAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req DeletePlaidAccountRequest

	if err := h.repository.DeletePlaidToken(ctx, req.UserID); err != nil {
		http.Error(w, "Failed to delete Plaid token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
