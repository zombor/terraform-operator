package terraform

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/zombor/terraform-operator/pkg/os"
)

func PlanWorkflow(ctx workflow.Context, input []string) (string, error) {
	var (
		planOut string

		err error
	)

	retrypolicy := &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        100 * time.Second,
		MaximumAttempts:        0, // unlimited retries
		NonRetryableErrorTypes: []string{},
	}

	id := workflow.GetInfo(ctx).WorkflowExecution.ID

	options := workflow.ActivityOptions{
		// Timeout options specify when to automatically timeout Activity functions.
		StartToCloseTimeout: time.Minute,
		// Optionally provide a customized RetryPolicy.
		// Temporal retries failed Activities by default.
		RetryPolicy: retrypolicy,
	}

	// Apply the options.
	ctx = workflow.WithActivityOptions(ctx, options)

	err = workflow.ExecuteActivity(ctx, os.Write, id, input).Get(ctx, nil)
	if err != nil {
		return "", err
	}

	err = workflow.ExecuteActivity(ctx, Plan, fmt.Sprintf("/tmp/%s", id)).Get(ctx, &planOut)
	if err != nil {
		return "", err
	}

	err = workflow.ExecuteActivity(ctx, os.Delete, id).Get(ctx, nil)

	return planOut, err
}

func ApplyWorkflow(ctx workflow.Context, input []string) (string, error) {
	var (
		out string

		err error
	)

	retrypolicy := &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        100 * time.Second,
		MaximumAttempts:        0, // unlimited retries
		NonRetryableErrorTypes: []string{},
	}

	id := workflow.GetInfo(ctx).WorkflowExecution.ID

	options := workflow.ActivityOptions{
		// Timeout options specify when to automatically timeout Activity functions.
		StartToCloseTimeout: time.Minute,
		// Optionally provide a customized RetryPolicy.
		// Temporal retries failed Activities by default.
		RetryPolicy: retrypolicy,
	}

	// Apply the options.
	ctx = workflow.WithActivityOptions(ctx, options)

	err = workflow.ExecuteActivity(ctx, os.Write, id, input).Get(ctx, nil)
	if err != nil {
		return "", err
	}

	err = workflow.ExecuteActivity(ctx, Apply, fmt.Sprintf("/tmp/%s", id)).Get(ctx, &out)
	if err != nil {
		return "", err
	}

	err = workflow.ExecuteActivity(ctx, os.Delete, id).Get(ctx, nil)

	return out, err
}
