package transaction

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"golang.org/x/crypto/ed25519"
)

type EstimateRequestPayload struct {
	Address      string `json:"address"`
	Body         string `json:"body"`
	IgnoreChksig bool   `json:"ignoreChksig"`
	InitCode     string `json:"initCode"`
	InitData     string `json:"initData"`
}

type EstimateResponsePayload struct {
	Ok     bool `json:"ok"`
	Result struct {
		SourceFees struct {
			InFwdFee   int64 `json:"in_fwd_fee"`
			StorageFee int64 `json:"storage_fee"`
			GasFee     int64 `json:"gas_fee"`
			FwdFee     int64 `json:"fwd_fee"`
		} `json:"source_fees"`
	} `json:"result"`
}

func GetTransactionBodyFromAPI(walletAddress string) (string, error) {

	url := fmt.Sprintf("https://toncenter.com/api/v2/getTransactions?address=%s&limit=1&to_lt=0&archival=false", walletAddress)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %s", err)
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get transaction: %s", resp.Status)
	}

	var respPayload struct {
		Result []struct {
			Body string `json:"body"`
		} `json:"result"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respPayload)
	if err != nil {
		return "", fmt.Errorf("error decoding response: %s", err)
	}

	if len(respPayload.Result) > 0 {
		return respPayload.Result[0].Body, nil
	}

	return "", nil
}

func SendTransaction(ctx context.Context, privateKey ed25519.PrivateKey, targetAddress string, amount string, sendUSDT bool, sendTON bool) {
	conn := liteclient.NewConnectionPool()
	mainnetConfigURL := "https://ton-blockchain.github.io/global.config.json"
	err := conn.AddConnectionsFromConfigUrl(ctx, mainnetConfigURL)
	if err != nil {
		log.Fatal(err)
	}

	api := ton.NewAPIClient(conn)

	w, err := wallet.FromPrivateKey(api, privateKey, wallet.V4R2)
	if err != nil {
		log.Fatal(err)
	}

	block, err := api.CurrentMasterchainInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}

	balance, err := w.GetBalance(ctx, block)
	if err != nil {
		log.Fatal(err)
	}

	parsedAmount, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	if balance.Nano().Uint64() < parsedAmount {
		fmt.Println("Insufficient balance. Do you want to try with a smaller amount? (yes/no): ")
		var tryAgainInput string
		fmt.Scanln(&tryAgainInput)

		if tryAgainInput != "yes" {
			fmt.Println("Transaction cancelled by user.")
			return
		} else {
			fmt.Println("Please enter a smaller amount:")
			var newAmount string
			fmt.Scanln(&newAmount)
			SendTransaction(ctx, privateKey, targetAddress, newAmount, sendUSDT, sendTON)
			return
		}
	}

	var userChoice string
	var walletAddress string

	fmt.Println("Do you want to use the default wallet address? (yes/no): ")
	fmt.Scanln(&userChoice)

	if userChoice == "no" {

		fmt.Println("Please enter the wallet address: ")
		fmt.Scanln(&walletAddress)
	} else if userChoice == "yes" {

		walletAddress = w.Address().String()
	} else {

		fmt.Println("Invalid choice. Using the default address.")
		walletAddress = w.Address().String()
	}

	addr, err := address.ParseAddr(walletAddress)
	if err != nil {
		log.Fatal(err)
	}

	body, err := GetTransactionBodyFromAPI(walletAddress)
	if err != nil {
		log.Fatal(err)
	}

	var messages []*wallet.Message
	if body == "" {

		if sendUSDT {
			bodyUSDT, _ := wallet.CreateCommentCell("Sending USDT")
			messages = append(messages, &wallet.Message{
				Mode: 1,
				InternalMessage: &tlb.InternalMessage{
					IHRDisabled: true,
					Bounce:      false,
					DstAddr:     addr,
					Amount:      tlb.FromNanoTONU(parsedAmount),
					Body:        bodyUSDT,
				},
			})
		}

		if sendTON {
			bodyTON, _ := wallet.CreateCommentCell("Sending TON")
			messages = append(messages, &wallet.Message{
				Mode: 1,
				InternalMessage: &tlb.InternalMessage{
					IHRDisabled: true,
					Bounce:      false,
					DstAddr:     addr,
					Amount:      tlb.FromNanoTONU(parsedAmount),
					Body:        bodyTON,
				},
			})
		}
	} else {

		fmt.Printf("Using the last transaction's body: %s\n", body)
		messages = append(messages, &wallet.Message{
			Mode: 1,
			InternalMessage: &tlb.InternalMessage{
				IHRDisabled: true,
				Bounce:      false,
				DstAddr:     addr,
				Body:        createBase64Body(body),
			},
		})
	}

	bocBytes := messages[0].InternalMessage.Body.ToBOC()
	if err != nil {
		log.Fatalf("error converting cell to BOC: %v", err)
	}

	bocBase64 := base64.StdEncoding.EncodeToString(bocBytes)
	fmt.Printf("Transaction body (base64): %s\n", bocBase64)

	fee, err := EstimateFee(walletAddress, bocBase64, "", "")
	if err != nil {
		log.Fatalf("error estimating fee: %v", err)
	}

	fmt.Printf("Estimated fee: %.9f TON. Do you want to proceed? (yes/no): ", fee)
	var userInput string
	fmt.Scanln(&userInput)

	if userInput != "yes" {
		fmt.Println("Transaction cancelled by user.")
		return
	}
}

func createBase64Body(body string) *cell.Cell {
	builder := cell.BeginCell()

	data := []byte(body)
	base64Data := base64.StdEncoding.EncodeToString(data)
	fmt.Printf("Data to be stored in cell (Base64 encoded): %s\n", base64Data)

	base64Bytes := []byte(base64Data)

	err := builder.StoreBinarySnake(base64Bytes)
	if err != nil {
		log.Fatalf("error storing Base64 data in cell: %v", err)
	}

	return builder.EndCell()
}
func EstimateFee(walletAddress, body, initCode, initData string) (float64, error) {

	base64Body := createBase64Body(body)

	base64BodyStr := base64.StdEncoding.EncodeToString(base64Body.ToBOC())

	requestPayload := EstimateRequestPayload{
		Address:      walletAddress,
		Body:         base64BodyStr,
		IgnoreChksig: true,
		InitCode:     initCode,
		InitData:     initData,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return 0, fmt.Errorf("error marshaling request: %s", err)
	}

	url := "https://toncenter.com/api/v2/estimateFee"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, fmt.Errorf("error creating request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	var respPayload EstimateResponsePayload
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get estimate fee: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&respPayload)
	if err != nil {
		return 0, fmt.Errorf("error decoding response: %s", err)
	}

	if !respPayload.Ok {
		return 0, fmt.Errorf("error estimating fee: invalid response")
	}

	totalFee := float64(respPayload.Result.SourceFees.InFwdFee +
		respPayload.Result.SourceFees.StorageFee +
		respPayload.Result.SourceFees.GasFee +
		respPayload.Result.SourceFees.FwdFee)

	totalFeeInNano := totalFee / 1000000000.0

	fmt.Printf("Total estimated fee: %.9f TON.\n", totalFeeInNano)

	return totalFeeInNano, nil
}
