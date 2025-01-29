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

// تابع ارسال درخواست به API و بررسی تراکنش‌ها
func GetTransactionBodyFromAPI(walletAddress string) (string, error) {
	// ساخت URL
	url := fmt.Sprintf("https://toncenter.com/api/v2/getTransactions?address=%s&limit=1&to_lt=0&archival=false", walletAddress)

	// ارسال درخواست GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %s", err)
	}

	// تعیین هدرها
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	// بررسی وضعیت پاسخ
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get transaction: %s", resp.Status)
	}

	var respPayload struct {
		Result []struct {
			Body string `json:"body"` // Body تراکنش در پاسخ
		} `json:"result"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respPayload)
	if err != nil {
		return "", fmt.Errorf("error decoding response: %s", err)
	}

	// اگر تراکنشی وجود داشته باشد، بادی آن را برمی‌گردانیم
	if len(respPayload.Result) > 0 {
		return respPayload.Result[0].Body, nil
	}

	// اگر تراکنشی وجود نداشته باشد
	return "", nil
}

// تابع ارسال تراکنش
// تابع ارسال تراکنش
func SendTransaction(ctx context.Context, privateKey ed25519.PrivateKey, targetAddress string, amount string, sendUSDT bool, sendTON bool) {
	conn := liteclient.NewConnectionPool()
	mainnetConfigURL := "https://ton-blockchain.github.io/global.config.json"
	err := conn.AddConnectionsFromConfigUrl(ctx, mainnetConfigURL)
	if err != nil {
		log.Fatal(err)
	}

	api := ton.NewAPIClient(conn)

	// ایجاد ولت از کلید خصوصی
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

	// Check balance
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

	// گرفتن آدرس ورودی از کاربر
	var userChoice string
	var walletAddress string

	// از کاربر بخواهید که انتخاب کند
	fmt.Println("Do you want to use the default wallet address? (yes/no): ")
	fmt.Scanln(&userChoice)

	if userChoice == "no" {
		// اگر کاربر آدرس خود را وارد کرده است
		fmt.Println("Please enter the wallet address: ")
		fmt.Scanln(&walletAddress)
	} else if userChoice == "yes" {
		// اگر کاربر می‌خواهد از آدرس پیش‌فرض استفاده کند
		walletAddress = w.Address().String()
	} else {
		// اگر ورودی کاربر معتبر نیست
		fmt.Println("Invalid choice. Using the default address.")
		walletAddress = w.Address().String()
	}

	// تبدیل آدرس به نوع Address
	addr, err := address.ParseAddr(walletAddress)
	if err != nil {
		log.Fatal(err)
	}

	// بررسی وضعیت تراکنش قبلی
	body, err := GetTransactionBodyFromAPI(walletAddress)
	if err != nil {
		log.Fatal(err)
	}

	var messages []*wallet.Message
	if body == "" {
		// اگر تراکنش قبلی وجود نداشت، بادی جدید بساز
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
		// اگر تراکنش قبلی وجود داشت، بادی تراکنش قبلی را استفاده کن
		fmt.Printf("Using the last transaction's body: %s\n", body) // نمایش بادی تراکنش قبلی
		messages = append(messages, &wallet.Message{
			Mode: 1,
			InternalMessage: &tlb.InternalMessage{
				IHRDisabled: true,
				Bounce:      false,
				DstAddr:     addr,
				Body:        createBase64Body(body), // استفاده از بادی تراکنش قبلی
			},
		})
	}

	// تبدیل سلول به BOC
	bocBytes := messages[0].InternalMessage.Body.ToBOC()
	if err != nil {
		log.Fatalf("error converting cell to BOC: %v", err)
	}

	bocBase64 := base64.StdEncoding.EncodeToString(bocBytes)
	fmt.Printf("Transaction body (base64): %s\n", bocBase64)

	// Estimate transaction fee
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

	// ارسال تراکنش
	txnHash, err := w.SendManyWaitTxHash(ctx, messages)
	if err != nil {
		log.Fatalf("error sending transaction: %v", err)
	}

	log.Println("Transaction(s) sent successfully. TXN HASH:", string(txnHash))
}

func createBase64Body(body string) *cell.Cell {
	builder := cell.BeginCell()

	// تبدیل داده‌ها به فرمت Base64
	data := []byte(body)
	base64Data := base64.StdEncoding.EncodeToString(data)
	fmt.Printf("Data to be stored in cell (Base64 encoded): %s\n", base64Data)

	// تبدیل رشته Base64 به []byte
	base64Bytes := []byte(base64Data)

	// ذخیره داده‌های Base64 به عنوان آرایه بایت در سلول
	err := builder.StoreBinarySnake(base64Bytes)
	if err != nil {
		log.Fatalf("error storing Base64 data in cell: %v", err)
	}

	// ساخت سلول و بازگشت آن
	return builder.EndCell()
}
func EstimateFee(walletAddress, body, initCode, initData string) (float64, error) {
	// تبدیل body به Base64 (اگر یک سلول باشد)
	base64Body := createBase64Body(body)

	// تبدیل سلول به Base64
	base64BodyStr := base64.StdEncoding.EncodeToString(base64Body.ToBOC())

	// ساخت Payload برای درخواست
	requestPayload := EstimateRequestPayload{
		Address:      walletAddress,
		Body:         base64BodyStr, // استفاده از بادی Base64 به جای بادی معمولی
		IgnoreChksig: true,
		InitCode:     initCode,
		InitData:     initData,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return 0, fmt.Errorf("error marshaling request: %s", err)
	}

	// استفاده از URL مناسب برای ارسال درخواست به API
	url := "https://toncenter.com/api/v2/estimateFee"

	// ارسال درخواست
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, fmt.Errorf("error creating request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// ایجاد یک client با timeout بیشتر
	client := &http.Client{Timeout: 60 * time.Second} // timeout را افزایش دهید
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	// پاسخ JSON
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

	// محاسبه هزینه‌ها
	totalFee := float64(respPayload.Result.SourceFees.InFwdFee +
		respPayload.Result.SourceFees.StorageFee +
		respPayload.Result.SourceFees.GasFee +
		respPayload.Result.SourceFees.FwdFee)

	// تبدیل به واحد TON (بر حسب نانو)
	totalFeeInNano := totalFee / 1000000000.0

	// نمایش هزینه تخمینی
	fmt.Printf("Total estimated fee: %.9f TON.\n", totalFeeInNano)

	return totalFeeInNano, nil
}
