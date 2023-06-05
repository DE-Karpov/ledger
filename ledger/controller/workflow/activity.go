package workflow

import (
	"context"
	"encore.app/ledger/controller/util"
	tb_types "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"log"
)

func logAccount(account *tb_types.Account) {
	log.Printf("Account ID: %s\n", account.ID)
	log.Printf("  - Credits Posted: %d\n", account.CreditsPosted)
	log.Printf("  - Debits Posted: %d\n", account.DebitsPosted)
	//TODO How to store them?
	//log.Printf("  - Available Balance: %d\n", account.CreditsPosted-account.DebitsPosted)
	log.Println()
}

func GetBalance(ctx context.Context, debitAccountID string) (interface{}, error) {
	accounts, err := util.TbClient.LookupAccounts([]tb_types.Uint128{util.Uint128OrPanic(debitAccountID)})
	if err != nil {
		log.Printf("Could not fetch accounts: %s\n", err)
		return new(interface{}), err
	}
	if len(accounts) == 0 {
		log.Printf("There are no accounts: %s\n", err)
		return new(interface{}), nil
	}
	account := accounts[0]
	log.Println("Account Details:")
	logAccount(&account)

	//TODO How to store?
	pureBalance := int(account.CreditsPosted) - int(account.DebitsPosted)
	debitsPending := int(account.DebitsPending)
	debitsPosted := int(account.DebitsPosted)
	creditsPending := int(account.CreditsPending)
	creditsPosted := int(account.CreditsPosted)
	withFreezedDebits := pureBalance + debitsPending
	withFreezedCredits := pureBalance + creditsPending
	totalDebits := debitsPending + int(account.DebitsPosted)
	totalCredits := creditsPending + int(account.CreditsPosted)
	totalBalance := totalCredits - totalDebits
	balance := struct {
		PureBalance        int
		DebitsPending      int
		DebitsPosted       int
		CreditsPending     int
		CreditsPosted      int
		WithFreezedDebits  int
		WithFreezedCredits int
		TotalDebits        int
		TotalCredits       int
		TotalBalance       int
	}{
		PureBalance:        pureBalance,
		DebitsPending:      debitsPending,
		DebitsPosted:       debitsPosted,
		CreditsPending:     creditsPending,
		CreditsPosted:      creditsPosted,
		WithFreezedDebits:  withFreezedDebits,
		WithFreezedCredits: withFreezedCredits,
		TotalDebits:        totalDebits,
		TotalCredits:       totalCredits,
		TotalBalance:       totalBalance,
	}

	log.Println("GetBalance completed successfully")
	return balance, nil
}

func FreezeFunds(ctx context.Context, debitAccountID, creditAccountID, amount string) (*Authorization, error) {
	log.Println("FreezeFunds called")

	transferID := util.GenerateUUID()

	_, err := util.TbClient.CreateTransfers([]tb_types.Transfer{
		{
			ID:              util.Uint128OrPanic(transferID),
			DebitAccountID:  util.Uint128OrPanic(debitAccountID),
			CreditAccountID: util.Uint128OrPanic(creditAccountID),
			Ledger:          1,
			Code:            1,
			Amount:          util.Uint64OrPanic(amount),
			Flags:           tb_types.TransferFlags{Pending: true}.ToUint16(),
		},
	})
	if err != nil {
		log.Printf("Could not create transfers: %s\n", err)
		return nil, err
	}

	transfers, err := util.TbClient.LookupTransfers([]tb_types.Uint128{util.Uint128OrPanic(transferID)})
	if err != nil || transfers == nil {
		log.Printf("Could not fetch transfers: %s\n", err)
		return nil, err
	}

	var authorization *Authorization
	for _, transfer := range transfers {
		authorization = &Authorization{
			ID:        transfer.ID.String(),
			AccountID: transfer.DebitAccountID.String(),
			Amount:    transfer.Amount,
			State:     "FREEZED",
		}
	}

	if err != nil {
		log.Printf("Error creating transfer: %s\n", err)
		return nil, err
	}

	log.Println("FreezeFunds completed successfully")
	return authorization, nil
}

func UnfreezeFunds(ctx context.Context, debitAccountID, creditAccountID, amount, freezedTransferID string) (bool, error) {
	log.Println("UnfreezeFunds called")

	transferID := util.GenerateUUID()

	_, err := util.TbClient.CreateTransfers([]tb_types.Transfer{
		{
			ID:              util.Uint128OrPanic(transferID),
			PendingID:       util.Uint128OrPanic(freezedTransferID),
			DebitAccountID:  util.Uint128OrPanic(debitAccountID),
			CreditAccountID: util.Uint128OrPanic(creditAccountID),
			Ledger:          1,
			Code:            1,
			Amount:          util.Uint64OrPanic(amount),
			Flags:           tb_types.TransferFlags{VoidPendingTransfer: true}.ToUint16(),
		},
	})
	if err != nil {
		log.Printf("Could not create transfers: %s\n", err)
		return false, err
	}

	transfers, err := util.TbClient.LookupTransfers([]tb_types.Uint128{util.Uint128OrPanic(transferID)})
	for _, transfer := range transfers {
		log.Printf("Transfer Details:")
		log.Printf("  - ID: %s\n", transfer.ID)
		log.Printf("  - PendingID: %d\n", transfer.PendingID)
		log.Printf("  - Debit Account ID: %s\n", transfer.DebitAccountID)
		log.Printf("  - Credit Account ID: %s\n", transfer.CreditAccountID)
		log.Printf("  - Ledger: %d\n", transfer.Ledger)
		log.Printf("  - Code: %d\n", transfer.Code)
		log.Printf("  - Flags: %d\n", transfer.Flags)
		log.Printf("  - Amount: %d\n", transfer.Amount)

		log.Println()
	}

	accounts, err := util.TbClient.LookupAccounts([]tb_types.Uint128{util.Uint128OrPanic(debitAccountID), util.Uint128OrPanic(creditAccountID)})
	if err != nil {
		log.Fatalf("Could not fetch accounts: %s\n", err)
		return false, err
	}

	if len(accounts) == 0 {
		log.Printf("There are no accounts found. If you see me - something's wrong with authorizations")
		return false, nil
	}

	for _, account := range accounts {
		log.Println("Account Details:")
		logAccount(&account)
	}

	log.Println("UnfreezeFunds completed successfully")
	return true, nil
}

func PostFunds(ctx context.Context, debitAccountID, creditAccountID, amount, freezedTransferID string) (bool, error) {
	log.Println("PostFunds called")

	accounts, err := util.TbClient.LookupAccounts([]tb_types.Uint128{util.Uint128OrPanic(debitAccountID), util.Uint128OrPanic(creditAccountID)})
	if err != nil {
		log.Fatalf("Could not fetch accounts: %s\n", err)
		return false, err
	}

	for _, account := range accounts {
		log.Println("Account Details:")
		logAccount(&account)
	}

	transferID := util.GenerateUUID()

	transferRes, err := util.TbClient.CreateTransfers([]tb_types.Transfer{
		{
			ID:              util.Uint128OrPanic(transferID),
			PendingID:       util.Uint128OrPanic(freezedTransferID),
			DebitAccountID:  util.Uint128OrPanic(debitAccountID),
			CreditAccountID: util.Uint128OrPanic(creditAccountID),
			Ledger:          1,
			Code:            1,
			Amount:          util.Uint64OrPanic(amount),
			Flags:           tb_types.TransferFlags{PostPendingTransfer: true}.ToUint16(),
		},
	})
	if err != nil {
		log.Fatalf("Error looking up transfers: %s\n", err)
		return false, err
	}

	for _, err := range transferRes {
		log.Fatalf("Error creating transfer: %s\n", err.Result)
	}

	transfers, err := util.TbClient.LookupTransfers([]tb_types.Uint128{util.Uint128OrPanic(debitAccountID), util.Uint128OrPanic("100")})
	if err != nil {
		log.Printf("Could not fetch transfers: %s\n", err)
		return false, err
	}

	for _, transfer := range transfers {
		log.Printf("Transfer Details:")
		log.Printf("  - ID: %s\n", transfer.ID)
		log.Printf("  - Debit Account ID: %s\n", transfer.DebitAccountID)
		log.Printf("  - Credit Account ID: %s\n", transfer.CreditAccountID)
		log.Printf("  - Ledger: %d\n", transfer.Ledger)
		log.Printf("  - Code: %d\n", transfer.Code)
		log.Printf("  - Amount: %d\n", transfer.Amount)
		log.Println()
	}

	accounts, err = util.TbClient.LookupAccounts([]tb_types.Uint128{util.Uint128OrPanic(debitAccountID), util.Uint128OrPanic(creditAccountID)})
	if err != nil {
		log.Fatalf("Could not fetch accounts: %s\n", err)
	}

	for _, account := range accounts {
		log.Println("Account Details:")
		logAccount(&account)
	}

	log.Println("PostFunds completed successfully")
	return true, nil
}
