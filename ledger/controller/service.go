package controller

import (
	"context"
	"encore.app/ledger/controller/workflow"
	encore "encore.dev"
	"fmt"
	tb "github.com/tigerbeetledb/tigerbeetle-go"
	tb_types "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
	"os"
)

var (
	envName                = encore.Meta().Environment.Name
	balanceTaskQueue       = envName + "-balance"
	authorizationTaskQueue = envName + "-authorize"
	presentmentTaskQueue   = envName + "-present"
)

//encore:service
type Service struct {
	Client  client.Client
	Workers []worker.Worker
}

type BalanceResponse struct {
	Available   int
	Freezed     int
	WithFreezed int
}

type PresentResponse struct {
	IsPresent bool
}

type AuthorizeResponse struct {
	Transferred bool
}

func uint128(value string) tb_types.Uint128 {
	x, err := tb_types.HexStringToUint128(value)
	if err != nil {
		panic(err)
	}
	return x
}

func initTigerbeetle() {
	TigerBeetleDBClient, err := tb.NewClient(0, []string{"3000"}, 32)
	if err != nil {
		log.Fatalf("Error creating TigerBeetleDBClient: %s", err)
	}
	defer TigerBeetleDBClient.Close()

	id1 := uint128("1")
	id2 := uint128("2")

	// Create two accounts
	res, err := TigerBeetleDBClient.CreateAccounts([]tb_types.Account{
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
		log.Fatalf("Error creating accounts: %s", err)
	}

	for _, err := range res {
		log.Fatalf("Error creating account %d: %s", err.Index, err.Result)
	}
}

func initService() (*Service, error) {
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("create temporal TigerBeetleDBClient: %v", err)
	}

	port := os.Getenv("TB_ADDRESS")
	if port == "" {
		port = "3000"
	}

	//initTigerbeetle()

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
	pw.RegisterActivity(workflow.Present)

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
		Client:  c,
		Workers: []worker.Worker{bw, aw, pw},
	}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.Client.Close()
	for _, localWorker := range s.Workers {
		localWorker.Stop()
	}
}
