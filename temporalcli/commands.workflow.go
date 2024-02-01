package temporalcli

import (
	"fmt"
	"os/user"

	"github.com/google/uuid"
	"github.com/temporalio/cli/temporalcli/internal/printer"
	"go.temporal.io/api/batch/v1"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

func (*TemporalWorkflowCancelCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (*TemporalWorkflowDeleteCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (*TemporalWorkflowQueryCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (*TemporalWorkflowResetCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (*TemporalWorkflowResetBatchCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (c *TemporalWorkflowSignalCommand) run(cctx *CommandContext, args []string) error {
	cl, err := c.Parent.ClientOptions.dialClient(cctx)
	if err != nil {
		return err
	}
	defer cl.Close()

	// Get input payloads
	input, err := c.buildRawInputPayloads()
	if err != nil {
		return err
	}

	exec, batchReq, err := c.workflowExecOrBatch(cctx, c.Parent.Namespace, cl)
	if err != nil {
		return err
	}

	// Run single or batch
	if exec != nil {
		// We have to use the raw signal service call here because the Go SDK's
		// signal call doesn't accept multiple arguments.
		_, err = cl.WorkflowService().SignalWorkflowExecution(cctx, &workflowservice.SignalWorkflowExecutionRequest{
			Namespace:         c.Parent.Namespace,
			WorkflowExecution: &common.WorkflowExecution{WorkflowId: c.WorkflowId, RunId: c.RunId},
			SignalName:        c.Name,
			Input:             input,
			Identity:          clientIdentity(),
		})
		if err != nil {
			return fmt.Errorf("failed signalling workflow: %w", err)
		}
		cctx.Printer.Println("Signal workflow succeeded")
	} else if batchReq != nil {
		batchReq.Operation = &workflowservice.StartBatchOperationRequest_SignalOperation{
			SignalOperation: &batch.BatchOperationSignal{
				Signal:   c.Name,
				Input:    input,
				Identity: clientIdentity(),
			},
		}
		if err := startBatchJob(cctx, cl, batchReq); err != nil {
			return err
		}
	}
	return nil
}

func (*TemporalWorkflowStackCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (*TemporalWorkflowTerminateCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (*TemporalWorkflowTraceCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (*TemporalWorkflowUpdateCommand) run(*CommandContext, []string) error {
	return fmt.Errorf("TODO")
}

func (s *SingleWorkflowOrBatchOptions) workflowExecOrBatch(
	cctx *CommandContext,
	namespace string,
	cl client.Client,
) (*common.WorkflowExecution, *workflowservice.StartBatchOperationRequest, error) {
	// If workflow is set, we return single execution
	if s.WorkflowId != "" {
		if s.Query != "" {
			return nil, nil, fmt.Errorf("cannot set query when workflow ID is set")
		} else if s.Reason != "" {
			return nil, nil, fmt.Errorf("cannot set reason when workflow ID is set")
		} else if s.Yes {
			return nil, nil, fmt.Errorf("cannot set 'yes' when workflow ID is set")
		}
		return &common.WorkflowExecution{WorkflowId: s.WorkflowId, RunId: s.RunId}, nil, nil
	}

	// Check query is set properly
	if s.Query == "" {
		return nil, nil, fmt.Errorf("must set either workflow ID or query")
	} else if s.WorkflowId != "" {
		return nil, nil, fmt.Errorf("cannot set workflow ID when query is set")
	} else if s.RunId != "" {
		return nil, nil, fmt.Errorf("cannot set run ID when query is set")
	}

	// Count the workflows that will be affected
	count, err := cl.CountWorkflow(cctx, &workflowservice.CountWorkflowExecutionsRequest{Query: s.Query})
	if err != nil {
		return nil, nil, fmt.Errorf("failed counting workflows from query: %w", err)
	}
	yes, err := cctx.promptYes(
		fmt.Sprintf("Start batch against approximately %v workflow(s)? y/N", count.Count), s.Yes)
	if err != nil {
		return nil, nil, err
	} else if !yes {
		// We consider this a command failure
		return nil, nil, fmt.Errorf("user denied confirmation")
	}

	// Default the reason if not set
	reason := s.Reason
	if reason == "" {
		username := "<unknown-user>"
		if u, err := user.Current(); err != nil && u.Username != "" {
			username = u.Username
		}
		reason = "Requested from CLI by " + username
	}

	return nil, &workflowservice.StartBatchOperationRequest{
		Namespace:       namespace,
		JobId:           uuid.NewString(),
		VisibilityQuery: s.Query,
		Reason:          reason,
	}, nil
}

func startBatchJob(cctx *CommandContext, cl client.Client, req *workflowservice.StartBatchOperationRequest) error {
	_, err := cl.WorkflowService().StartBatchOperation(cctx, req)
	if err != nil {
		return fmt.Errorf("failed starting batch operation: %w", err)
	}
	if cctx.JSONOutput {
		return cctx.Printer.PrintStructured(
			struct {
				BatchJobID string `json:"batchJobId"`
			}{BatchJobID: req.JobId},
			printer.StructuredOptions{})
	}
	cctx.Printer.Printlnf("Started batch for job ID: %v", req.JobId)
	return nil
}