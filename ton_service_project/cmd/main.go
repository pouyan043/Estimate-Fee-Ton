package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"ton_service_project/transaction"
	"ton_service_project/utils"
)

func main() {
	// نوع تراکنش: TON یا USDT
	var transactionType string
	fmt.Println("Do you want to send TON or USDT? (Enter 'TON' or 'USDT'): ")
	fmt.Scanln(&transactionType)

	transactionType = strings.ToLower(transactionType)

	var isUSDT bool
	if transactionType == "usdt" {
		isUSDT = true
	} else if transactionType == "ton" {
		isUSDT = false
	} else {
		fmt.Println("Invalid input. Please enter either 'TON' or 'USDT'.")
		return
	}

	// بارگذاری اطلاعات کیف پول از فایل .env یا تولید داده‌های جدید
	var publicKey, privateKey, address, mnemonic, seed string
	if _, err := os.Stat(".env"); err == nil {
		// اگر فایل .env وجود داشته باشد، داده‌ها را بارگذاری می‌کنیم
		fmt.Println(".env file found, loading wallet data...")
		utils.LoadEnvData(&publicKey, &privateKey, &address, &mnemonic, &seed)
	} else {
		// اگر فایل .env وجود نداشته باشد، داده‌های جدید تولید می‌کنیم
		fmt.Println(".env file not found, generating new wallet data...")
		publicKey, privateKey, address, mnemonic, seed = utils.GenerateWalletData()
		utils.SaveToEnvFile(publicKey, privateKey, address, mnemonic, seed)
	}

	// پرسش از کاربر: آیا می‌خواهید از آدرس پیش‌فرض استفاده کنید؟
	var userChoice string
	fmt.Print("Do you want to use the default wallet address? (yes/no): ")
	fmt.Scanln(&userChoice)

	var addressToUse string
	if userChoice == "yes" {

		addressToUse = address
	} else {

		fmt.Print("Please enter the wallet address: ")
		fmt.Scanln(&addressToUse)

		if addressToUse == "" {
			log.Fatal("Wallet address cannot be empty")
		}
	}

	privateKeyBytes, err := utils.DecodeBase64WithPadding(privateKey)
	if err != nil {
		log.Fatal("Failed to decode private key:", err)
	}

	ctx := context.Background()
	var sendTON, sendUSDT bool
	if isUSDT {
		sendUSDT = true
		sendTON = false
	} else {
		sendTON = true
		sendUSDT = false
	}

	var body string
	if sendUSDT {
		body = "Sending USDT"
	} else if sendTON {
		body = "Sending TON"
	}

	initCode := ""
	initData := ""

	fee, err := transaction.EstimateFee(addressToUse, body, initCode, initData)
	if err != nil {
		log.Fatal("Error estimating fee:", err)
	}
	fmt.Printf("Estimated fee: %.6f TON\n", fee)

	// ارسال تراکنش
	transaction.SendTransaction(ctx, privateKeyBytes, addressToUse, "1000000", sendUSDT, sendTON)
}
