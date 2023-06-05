package controller

import (
	"context"
	"encore.app/ledger/controller/util"
	"encore.app/ledger/controller/workflow"
	"encore.dev/rlog"
	"fmt"
	"go.temporal.io/sdk/client"
	"log"
)

// Balance retrieves the balance for an account.
//
//encore:api public path=/balance/:debitAccountID
func (s *Service) Balance(ctx context.Context, debitAccountID string) (*util.BalanceResponse, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        util.GenerateUUID() + "-balance-workflow",
		TaskQueue: balanceTaskQueue,
	}
	we, err := s.WorkflowClient.ExecuteWorkflow(ctx, workflowOptions, workflow.BalanceFlow, debitAccountID)
	if err != nil {
		return nil, err
	}
	rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

	var result *util.BalanceResponse
	err = we.Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	fmt.Println("Response: ", result)
	return result, nil
}

// Authorize authorizes an account.
//
//encore:api public path=/authorize/:debitAccountID/:creditAccountID/:amount
func (s *Service) Authorize(ctx context.Context, debitAccountID, creditAccountID, amount string) (*util.AuthorizeResponse, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        util.GenerateUUID() + "-authorization-workflow",
		TaskQueue: authorizationTaskQueue,
	}
	we, err := s.WorkflowClient.ExecuteWorkflow(ctx, workflowOptions, workflow.AuthorizationFlow, debitAccountID, creditAccountID, amount)
	if err != nil {
		return nil, err
	}
	rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

	var result bool
	err = we.Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &util.AuthorizeResponse{Transferred: result}, nil
}

// Present presents an account.
//
//encore:api public path=/present/:debitAccountID/:amount
func (s *Service) Present(ctx context.Context, debitAccountID, amount string) (*util.PresentResponse, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        util.GenerateUUID() + "-present-workflow",
		TaskQueue: presentmentTaskQueue,
	}
	we, err := s.WorkflowClient.ExecuteWorkflow(ctx, workflowOptions, workflow.PresentmentFlow, debitAccountID, amount)
	if err != nil {
		return nil, err
	}
	rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

	var response bool
	err = we.Get(ctx, &response)
	if err != nil {
		return nil, err
	}

	log.Println(response)
	return &util.PresentResponse{IsPresent: response}, nil
}
