package workflow

import (
	"context"
	"fmt"
	tb "github.com/tigerbeetledb/tigerbeetle-go"
	tb_types "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"log"
	"math/rand"
	"strconv"
	"time"
)

func uint128(value string) tb_types.Uint128 {
	x, err := tb_types.HexStringToUint128(value)
	if err != nil {
		panic(err)
	}
	return x
}

func fromStringToUint64(str string) uint64 {
	num, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}
	return num
}

func logAccount(account *tb_types.Account) {
	log.Printf("Account ID: %s\n", account.ID)
	log.Printf("  - Credits Posted: %d\n", account.CreditsPosted)
	log.Printf("  - Debits Posted: %d\n", account.DebitsPosted)
	//TODO How to store them?
	//log.Printf("  - Available Balance: %d\n", account.CreditsPosted-account.DebitsPosted)
	log.Println()
}

func generateUniqueID() string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	randomNum := rand.Intn(10000)

	uniqueID := fmt.Sprintf("%d%d", timestamp, randomNum)
	return uniqueID
}

func GetBalance(ctx context.Context, debitAccountID string) (*BalanceResponse, error) {
	client, err := tb.NewClient(0, []string{"3000"}, 32)
	if err != nil {
		log.Printf("Error creating client: %s\n", err)
		return new(BalanceResponse), err
	}
	defer client.Close()

	accounts, err := client.LookupAccounts([]tb_types.Uint128{uint128(debitAccountID)})
	if err != nil {
		log.Printf("Could not fetch accounts: %s\n", err)
		return new(BalanceResponse), err
	}

	account := accounts[0]
	log.Println("Account Details:")
	logAccount(&account)

	//TODO How to store?
	available := int(account.CreditsPosted) - int(account.DebitsPosted)
	freezed := int(account.DebitsPending)
	withFreezed := available + freezed
	balance := &BalanceResponse{Available: available, WithFreezed: withFreezed}

	log.Println("GetBalance completed successfully")
	return balance, nil
}

func FreezeFunds(ctx context.Context, debitAccountID, creditAccountID, amount string) (*Authorization, error) {
	client, err := tb.NewClient(0, []string{"3000"}, 32)
	if err != nil {
		log.Printf("Error creating client: %s\n", err)
		return nil, err
	}
	defer client.Close()

	log.Println("FreezeFunds called")

	transferID := generateUniqueID()

	transferRes, err := client.CreateTransfers([]tb_types.Transfer{
		{
			ID:              uint128(transferID),
			DebitAccountID:  uint128(debitAccountID),
			CreditAccountID: uint128(creditAccountID),
			Ledger:          1,
			Code:            1,
			Amount:          fromStringToUint64(amount),
			Flags:           tb_types.TransferFlags{Pending: true}.ToUint16(),
		},
	})
	if err != nil {
		log.Printf("Could not create transfers: %s\n", err)
		return nil, err
	}

	transfers, err := client.LookupTransfers([]tb_types.Uint128{uint128(transferID)})
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

	for _, err := range transferRes {
		log.Printf("Error creating transfer: %s\n", err.Result)
	}

	log.Println("FreezeFunds completed successfully")
	return authorization, nil
}

func UnfreezeFunds(ctx context.Context, debitAccountID, creditAccountID, amount, freezedTransferID string) (bool, error) {
	client, err := tb.NewClient(0, []string{"3000"}, 32)
	if err != nil {
		log.Printf("Error creating client: %s\n", err)
		return false, err
	}
	defer client.Close()

	log.Println("UnfreezeFunds called")

	transferID := generateUniqueID()

	transferRes, err := client.CreateTransfers([]tb_types.Transfer{
		{
			ID:              uint128(transferID),
			PendingID:       uint128(freezedTransferID),
			DebitAccountID:  uint128(debitAccountID),
			CreditAccountID: uint128(creditAccountID),
			Ledger:          1,
			Code:            1,
			Amount:          fromStringToUint64(amount),
			Flags:           tb_types.TransferFlags{VoidPendingTransfer: true}.ToUint16(),
		},
	})
	if err != nil {
		log.Printf("Could not create transfers: %s\n", err)
		return false, err
	}

	transfers, err := client.LookupTransfers([]tb_types.Uint128{uint128(transferID)})
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

	for _, err := range transferRes {
		// TODO What???
		log.Printf("Error creating transfer: %s\n", err.Result)
	}

	accounts, err := client.LookupAccounts([]tb_types.Uint128{uint128(debitAccountID), uint128(creditAccountID)})
	if err != nil {
		log.Fatalf("Could not fetch accounts: %s\n", err)
		return false, err
	}

	for _, account := range accounts {
		log.Println("Account Details:")
		logAccount(&account)
	}

	log.Println("UnfreezeFunds completed successfully")
	return true, nil
}

func Present(ctx context.Context, amount, authId string) (bool, error) {
	client, err := tb.NewClient(0, []string{"3000"}, 32)
	if err != nil {
		log.Printf("Error creating client: %s\n", err)
		return false, err
	}
	defer client.Close()

	log.Println("Present called")

	transfers, err := client.LookupTransfers([]tb_types.Uint128{uint128(authId)})
	if err != nil {
		log.Fatalf("Error looking up transfers: %s\n", err)
		return false, err
	}

	transfer := transfers[0]
	if uint64ToString(transfer.Amount) == amount && transfer.TransferFlags().Pending {
		log.Printf("Found a pending transfer: %s", transfer.ID)
		return true, nil
	}
	return false, nil
}

func PostFunds(ctx context.Context, debitAccountID, creditAccountID, amount, freezedTransferID string) (bool, error) {
	client, err := tb.NewClient(0, []string{"3000"}, 32)
	if err != nil {
		log.Printf("Error creating client: %s\n", err)
		return false, err
	}
	defer client.Close()

	log.Println("PostFunds called")

	accounts, err := client.LookupAccounts([]tb_types.Uint128{uint128(debitAccountID), uint128(creditAccountID)})
	if err != nil {
		log.Fatalf("Could not fetch accounts: %s\n", err)
		return false, err
	}

	for _, account := range accounts {
		log.Println("Account Details:")
		logAccount(&account)
	}

	transferID := generateUniqueID()

	transferRes, err := client.CreateTransfers([]tb_types.Transfer{
		{
			ID:              uint128(transferID),
			PendingID:       uint128(freezedTransferID),
			DebitAccountID:  uint128(debitAccountID),
			CreditAccountID: uint128(creditAccountID),
			Ledger:          1,
			Code:            1,
			Amount:          fromStringToUint64(amount),
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

	transfers, err := client.LookupTransfers([]tb_types.Uint128{uint128(debitAccountID), uint128("100")})
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

	accounts, err = client.LookupAccounts([]tb_types.Uint128{uint128(debitAccountID), uint128(creditAccountID)})
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
