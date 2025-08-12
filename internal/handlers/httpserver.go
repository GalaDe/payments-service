package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GalaDe/payments-service/internal/domain"
	"github.com/GalaDe/payments-service/internal/services/plaid"
	"github.com/GalaDe/payments-service/internal/services/stripe"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type HttpServer struct {
	logger        *zap.Logger
	worker        client.Client
	repository    domain.Repository
	plaidService  plaid.PlaidService
	stripeService stripe.StripeService
}

func NewHttpServer(logger *zap.Logger, worker client.Client, repository domain.Repository,
	plaidService plaid.PlaidService, stripeService stripe.StripeService) *HttpServer {
	return &HttpServer{
		logger:        logger,
		worker:        worker,
		repository:    repository,
		plaidService:  plaidService,
		stripeService: stripeService,
	}
}

func (h *HttpServer) respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func (h *HttpServer) respondWithError(w http.ResponseWriter, status int, message string) {
	h.respondWithJSON(w, status, map[string]string{"error": message})
}
