package workflow

import (
	"encore.app/ledger/controller/util"
	"log"
	"sort"
	"strconv"
	"time"

	"go.temporal.io/sdk/workflow"
)

type Authorization struct {
	ID         string
	WorkflowID string
	AccountID  string
	Amount     uint64
	State      string
	Timestamp  uint64
}

// TODO another DB?
type Authorizations []Authorization

func (a Authorizations) Len() int {
	return len(a)
}

func (a Authorizations) Less(i, j int) bool {
	return a[i].Timestamp < a[j].Timestamp
}

func (a Authorizations) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

var Auths Authorizations

func BalanceFlow(ctx workflow.Context, accountID string) (*util.BalanceResponse, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 5,
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var result *util.BalanceResponse
	err := workflow.ExecuteActivity(ctx, GetBalance, accountID).Get(ctx, &result)
	if err != nil {
		workflow.GetLogger(ctx).Error("Failed to execute GetBalance activity.", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("BalanceFlow completed successfully.")
	return result, nil
}

func AuthorizationFlow(ctx workflow.Context, debitAccountID, creditAccountID, amount string) (bool, error) {
	options := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Second,
	}

	var processingDone bool

	ctx = workflow.WithActivityOptions(ctx, options)
	childCtx, cancelHandler := workflow.WithCancel(ctx)
	signalChan := workflow.GetSignalChannel(ctx, "presentmentMatched_"+debitAccountID)
	selector := workflow.NewSelector(ctx)

	var freezedAuthorization Authorization
	err := workflow.ExecuteActivity(ctx, FreezeFunds, debitAccountID, creditAccountID, amount).Get(ctx, &freezedAuthorization)
	if err != nil {
		workflow.GetLogger(ctx).Error("Failed to execute FreezeFunds activity.", "error", err)
		return false, err
	}
	freezedAuthorization.WorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID

	Auths = append(Auths, freezedAuthorization)
	for _, auth := range Auths {
		log.Println(auth)
	}

	workflow.GetLogger(ctx).Info("FreezeFunds activity executed successfully.")

	var matchedAuth Authorization
	selector.AddReceive(signalChan, func(channel workflow.ReceiveChannel, more bool) {
		channel.Receive(ctx, &matchedAuth)
		defer func() {
			deleteAuthorization(&matchedAuth)
			processingDone = true
			cancelHandler()
		}()
		workflow.GetLogger(ctx).Info("Received presentmentMatched signal.")

		err := workflow.ExecuteActivity(ctx, PostFunds, debitAccountID, creditAccountID, amount, matchedAuth.ID).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to execute PostFunds activity.", "error", err)
		} else {
			workflow.GetLogger(ctx).Info("PostFunds activity executed successfully.")
		}

	})

	timerFuture := workflow.NewTimer(childCtx, 100*time.Second)
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		if !processingDone {
			defer deleteAuthorization(&freezedAuthorization)
			err := workflow.ExecuteActivity(ctx, UnfreezeFunds, debitAccountID, creditAccountID, amount, freezedAuthorization.ID).Get(ctx, freezedAuthorization)
			if err != nil {
				workflow.GetLogger(ctx).Error("Failed to execute UnfreezeFunds activity.", "error", err)
			} else {
				workflow.GetLogger(ctx).Info("UnfreezeFunds activity executed successfully.")
			}

		}
	})

	selector.Select(childCtx)

	workflow.GetLogger(ctx).Info("AuthorizationFlow completed successfully.")
	return processingDone, nil
}

func PresentmentFlow(ctx workflow.Context, debitAccountID, amount string) (bool, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	intAmount, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		return false, err
	}

	workflow.GetLogger(ctx).Info("Int amount in presentment flow")
	workflow.GetLogger(ctx).Info(amount)

	var authorizations Authorizations
	for _, auth := range Auths {
		if auth.AccountID == debitAccountID && auth.Amount == intAmount && auth.State == "FREEZED" {
			authorizations = append(authorizations, auth)
		}
	}

	log.Println(authorizations)
	if len(authorizations) == 0 {
		return false, nil
	}

	sort.Sort(authorizations)

	workflow.GetLogger(ctx).Info("Sending signal to authorization workflow.")
	auth := authorizations[0]
	err = workflow.SignalExternalWorkflow(ctx, auth.WorkflowID, "", "presentmentMatched_"+debitAccountID, auth).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("Failed to send signal to authorization workflow.", "error", err)
		return false, err
	}
	workflow.GetLogger(ctx).Info("Signal sent to authorization workflow.")

	workflow.GetLogger(ctx).Info("PresentmentFlow completed successfully.")
	return true, nil
}

func deleteAuthorization(auth *Authorization) {
	for i, a := range Auths {
		if a.ID == auth.ID {
			Auths = append(Auths[:i], Auths[i+1:]...)
			break
		}
	}
}
