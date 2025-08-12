package workflow

import (
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	PaymentWorkflow = "PaymentWorkflow"
)

func RegisterWorkflows(c worker.WorkflowRegistry) {
	c.RegisterWorkflowWithOptions(paymentWorkflow, workflow.RegisterOptions{Name: PaymentWorkflow})
}
