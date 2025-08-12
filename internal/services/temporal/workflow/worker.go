package workflow

import (
	"context"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
)

type ContextKey string

const (
	DefaultActivityTimeout = 120 * time.Second
	DefaultTaskQueue       = "default-task-queue"
	ClientContextKey ContextKey = "Client"
)

var (
	// import activities interface for calling activities from Workflows
	DefaultRetryPolicy = &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Second * 100,
	}

	RetryPolicy1Attempt = &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Second * 100,
		MaximumAttempts:    1,
	}

	RetryPolicy3Attempts = &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	}
)

type TemporalConfig struct {
	AccountID          string `json:"ACCOUNT_ID"`
	NamespaceID        string `json:"NAMESPACE_ID"`
	HostURI            string `json:"HOST_URI"`
	HostBaseURI        string `json:"HOST_BASE_URI"`
	HostPort           int    `json:"HOST_PORT"`
	CertPem            string `json:"CERT_PEM"`
	CertKey            string `json:"CERT_KEY"`
	PrivateLinkBaseURI string `json:"PRIVATE_LINK_BASE_URI"`
	PrivateLinkURI     string `json:"PRIVATE_LINK_URI"`
}

func NewWorker(t client.Client) worker.Worker {
	return worker.New(t, DefaultTaskQueue, worker.Options{
		BackgroundActivityContext:        context.WithValue(context.Background(), ClientContextKey, t),
		MaxConcurrentActivityTaskPollers: 8, // Default is 2
		MaxConcurrentWorkflowTaskPollers: 8, // Default is 2
	})
}
