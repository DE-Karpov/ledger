package util

import (
	"fmt"
	tb "github.com/tigerbeetledb/tigerbeetle-go"
	tb_types "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

type BalanceResponse struct {
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
}

type PresentResponse struct {
	IsPresent bool
}

type AuthorizeResponse struct {
	Transferred bool
}

var TbClient tb.Client

func init() {
	port := os.Getenv("TB_ADDRESS")
	if port == "" {
		port = "3000"
	}
	var err error
	TbClient, err = tb.NewClient(0, []string{port}, 32)
	if err != nil {
		log.Fatalf("Error creating TigerBeetleDBClient: %s", err)
	}
}

func Uint128OrPanic(value string) tb_types.Uint128 {
	x, err := tb_types.HexStringToUint128(value)
	if err != nil {
		panic(err)
	}
	return x
}

func Uint64OrPanic(str string) uint64 {
	num, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		panic(err)
	}
	return num
}

func GenerateUUID() string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	randomNum := rand.Intn(10000)

	uniqueID := fmt.Sprintf("%d%d", timestamp, randomNum)
	return uniqueID
}
