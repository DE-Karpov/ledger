package controller

import (
	"context"
	"encore.app/ledger/controller/workflow"
	"encore.dev/rlog"
	"fmt"
	"go.temporal.io/sdk/client"
	"log"
	"math/rand"
	"time"
)

func generateUniqueID() string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	randomNum := rand.Intn(10000)

	uniqueID := fmt.Sprintf("%d%d", timestamp, randomNum)
	return uniqueID
}

// Balance retrieves the balance for an account.
//
//encore:api public path=/balance/:debitAccountID
func (s *Service) Balance(ctx context.Context, debitAccountID string) (*BalanceResponse, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        generateUniqueID() + "-balance-workflow",
		TaskQueue: balanceTaskQueue,
	}
	we, err := s.Client.ExecuteWorkflow(ctx, workflowOptions, workflow.BalanceFlow, debitAccountID)
	if err != nil {
		return nil, err
	}
	rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

	var result *BalanceResponse
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
func (s *Service) Authorize(ctx context.Context, debitAccountID, creditAccountID, amount string) (*AuthorizeResponse, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        generateUniqueID() + "-authorization-workflow",
		TaskQueue: authorizationTaskQueue,
	}
	we, err := s.Client.ExecuteWorkflow(ctx, workflowOptions, workflow.AuthorizationFlow, debitAccountID, creditAccountID, amount)
	if err != nil {
		return nil, err
	}
	rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

	var result bool
	err = we.Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &AuthorizeResponse{Transferred: result}, nil
}

// Present presents an account.
//
//encore:api public path=/present/:debitAccountID/:amount
func (s *Service) Present(ctx context.Context, debitAccountID, amount string) (*PresentResponse, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        generateUniqueID() + "-present-workflow",
		TaskQueue: presentmentTaskQueue,
	}
	we, err := s.Client.ExecuteWorkflow(ctx, workflowOptions, workflow.PresentmentFlow, debitAccountID, amount)
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
	return &PresentResponse{IsPresent: response}, nil
}
