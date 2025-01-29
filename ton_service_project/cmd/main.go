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

	var publicKey, privateKey, address, mnemonic, seed string
	if _, err := os.Stat(".env"); err == nil {

		fmt.Println(".env file found, loading wallet data...")
		utils.LoadEnvData(&publicKey, &privateKey, &address, &mnemonic, &seed)
	} else {

		fmt.Println(".env file not found, generating new wallet data...")
		publicKey, privateKey, address, mnemonic, seed = utils.GenerateWalletData()
		utils.SaveToEnvFile(publicKey, privateKey, address, mnemonic, seed)
	}

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

	transaction.SendTransaction(ctx, privateKeyBytes, addressToUse, "1000000", sendUSDT, sendTON)
}
