package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	config "github.com/GalaDe/payments-service/internal/config"
	handler "github.com/GalaDe/payments-service/internal/handlers"
	plaid "github.com/GalaDe/payments-service/internal/services/plaid"
	stripe "github.com/GalaDe/payments-service/internal/services/stripe"
	"github.com/GalaDe/payments-service/internal/services/temporal"
	"github.com/GalaDe/payments-service/internal/services/temporal/activity"
	"github.com/GalaDe/payments-service/internal/services/temporal/workflow"
	orm "github.com/GalaDe/payments-service/internal/sqlc"
	repository "github.com/GalaDe/payments-service/internal/storage/postgres"
)

func main() {

	// Load configuration (e.g. from env or file)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initializing logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync() // flush logs before exiting

	// Setup DB connection
	pgSecret := &repository.PostgresSecret{
		DBConnString: cfg.DatabaseURL,
	}

	db, err := repository.NewPostgresDB(pgSecret)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	transactor := repository.NewPostgresTransactor(db, orm.New(db.Pool))
	repo := repository.NewPostgresRepo(transactor)

	defer db.Close()

	clientOptions := client.Options{
		HostPort: client.DefaultHostPort,
		Logger:   temporal.NewZapAdapter(logger),
	}

	temporalClient, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatalf("unable to create Temporal client: %v", err)
	}
	defer temporalClient.Close()

	w := workflow.NewWorker(temporalClient)
	workflow.RegisterWorkflows(w)

	// Build repository, services, and handlers
	stripeConfig := &stripe.StripeConfig{}
	stripeSvc := stripe.NewStripe(stripeConfig)
	plaidOpt := &plaid.PlaidOpts{}
	plaidSvc := plaid.New(plaidOpt)

	activityPort := activity.NewTemporalActivityPort(repo, stripeSvc, plaidSvc, temporalClient)
	activityPort.RegisterActivities(w)

	httpHandler := handler.NewHttpServer(logger, temporalClient, repo, plaidSvc, stripeSvc)

	// HTTP router
	r := chi.NewRouter()

	// (optional) CORS
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // Change for production
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	})
	r.Use(corsMiddleware.Handler)

	// Health check
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("✅ Go backend is up!"))
	})

	// Register your routes
	r = handler.RegisterRoutes(httpHandler)
	http.ListenAndServe(":8081", r)

	// Start server
	port := cfg.Port
	fmt.Printf("🚀 Listening on :%s...\n", port)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
