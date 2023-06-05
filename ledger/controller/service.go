package controller

import (
	"context"
	"encore.app/ledger/controller/util"
	"encore.app/ledger/controller/workflow"
	encore "encore.dev"
	"fmt"
	tb_types "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
)

var (
	envName                = encore.Meta().Environment.Name
	balanceTaskQueue       = envName + "-balance"
	authorizationTaskQueue = envName + "-authorize"
	presentmentTaskQueue   = envName + "-present"
)

//encore:service
type Service struct {
	WorkflowClient client.Client
	Workers        []worker.Worker
}

func initTigerbeetle() error {
	id1 := util.Uint128OrPanic("1")
	id2 := util.Uint128OrPanic("2")

	_, err := util.TbClient.CreateAccounts([]tb_types.Account{
		{
			ID:     id1,
			Ledger: 1,
			Code:   1,
		},
		{
			ID:     id2,
			Ledger: 1,
			Code:   1,
		},
	})
	if err != nil {
		log.Printf("Error creating accounts: %s", err)
		return err
	}

	return nil
}

func initService() (*Service, error) {
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("create temporal TigerBeetleDBClient: %v", err)
	}

	err = initTigerbeetle()

	if err != nil {
		log.Fatal("TbClient hasn't been initialized")
		return nil, err
	}

	bw := worker.New(c, balanceTaskQueue, worker.Options{})
	aw := worker.New(c, authorizationTaskQueue, worker.Options{})
	pw := worker.New(c, presentmentTaskQueue, worker.Options{})

	bw.RegisterWorkflow(workflow.BalanceFlow)
	aw.RegisterWorkflow(workflow.AuthorizationFlow)
	pw.RegisterWorkflow(workflow.PresentmentFlow)

	bw.RegisterActivity(workflow.GetBalance)
	aw.RegisterActivity(workflow.FreezeFunds)
	aw.RegisterActivity(workflow.UnfreezeFunds)
	aw.RegisterActivity(workflow.PostFunds)

	if err := bw.Start(); err != nil {
		c.Close()
		return nil, fmt.Errorf("start temporal worker: %v", err)
	}
	if err := aw.Start(); err != nil {
		c.Close()
		return nil, fmt.Errorf("start temporal worker: %v", err)
	}
	if err := pw.Start(); err != nil {
		c.Close()
		return nil, fmt.Errorf("start temporal worker: %v", err)
	}

	return &Service{
		WorkflowClient: c,
		Workers:        []worker.Worker{bw, aw, pw},
	}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.WorkflowClient.Close()
	for _, localWorker := range s.Workers {
		localWorker.Stop()
	}
	util.TbClient.Close()
}
