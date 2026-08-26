package temporalruntime

import (
	"context"
	"time"

	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const DevelopmentTaskQueue = "documi-development"

type DevelopmentInput struct {
	Message string
}

type DevelopmentResult struct {
	Echoed string
}

func DevelopmentWorkflow(ctx workflow.Context, input DevelopmentInput) (DevelopmentResult, error) {
	options := workflow.ActivityOptions{StartToCloseTimeout: 30 * time.Second}
	ctx = workflow.WithActivityOptions(ctx, options)
	var result DevelopmentResult
	err := workflow.ExecuteActivity(ctx, DevelopmentActivity, input).Get(ctx, &result)
	return result, err
}

func DevelopmentActivity(_ context.Context, input DevelopmentInput) (DevelopmentResult, error) {
	return DevelopmentResult{Echoed: input.Message}, nil
}

func RegisterDevelopment(registry worker.Registry) {
	registry.RegisterWorkflow(DevelopmentWorkflow)
	registry.RegisterActivity(DevelopmentActivity)
}
