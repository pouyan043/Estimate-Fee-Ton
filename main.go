package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// TransactionParams struct to hold the transaction parameters
type TransactionParams struct {
	Address  string `json:"address"`   // Address to send
	Body     string `json:"body"`      // Body of the transaction
	Value    int64  `json:"value"`     // Amount in nanograms (1 TON = 1e9 nanograms)
	GasPrice int64  `json:"gas_price"` // Gas price in nanograms
	GasLimit int64  `json:"gas_limit"` // Gas limit
}

// Fees struct to hold the fees information
type Fees struct {
	InFwdFee   int64 `json:"in_fwd_fee"`  // Inward forward fee
	StorageFee int64 `json:"storage_fee"` // Storage fee
	GasFee     int64 `json:"gas_fee"`     // Gas fee
	FwdFee     int64 `json:"fwd_fee"`     // Forward fee
}

// EstimatedFeeResponse struct to hold the response from the API
type EstimatedFeeResponse struct {
	Ok     bool `json:"ok"` // Response status
	Result struct {
		SourceFees Fees `json:"source_fees"` // Source fees
	} `json:"result"`
	Error string `json:"error"` // Error message if any
}

func main() {
	// Define the parameters for the transaction
	transactionParams := TransactionParams{
		Address:  "recipient address", // Replace with the recipient address
		Body:     "te6ccgEBAQEAAgAAAA==",                             // Body of the transaction in base64 format
		Value:    1000000000,                                         // Amount in nanograms (1 TON = 1e9 nanograms)
		GasPrice: 1000000000,                                         // Gas price in nanograms
		GasLimit: 2000000,                                            // Gas limit
	}

	// Convert the transaction parameters to JSON
	jsonData, err := json.Marshal(transactionParams)
	if err != nil {
		log.Fatalf("Failed to marshal transaction parameters: %v", err)
	}

	// Create a custom HTTP client with increased timeout and TLS configuration
	client := &http.Client{
		Timeout: time.Second * 30, // Increase the timeout to 30 seconds
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Make the HTTP request to the public API
	apiURL := "https://toncenter.com/api/v2/estimateFee" // Replace with your public API endpoint
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to create the HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer YOUR_API_KEY") // Replace YOUR_API_KEY with your actual API key

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to make the HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read and process the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read the response body: %v", err)
	}

	// Print the raw response body for debugging purposes
	fmt.Println("Raw response body:", string(body))

	var estimatedFeeResponse EstimatedFeeResponse
	err = json.Unmarshal(body, &estimatedFeeResponse)
	if err != nil {
		log.Fatalf("Failed to unmarshal the response body: %v", err)
	}

	// Check if there is an error message in the response
	if estimatedFeeResponse.Ok {
		// Print the estimated fees
		fmt.Printf("Inward Forward Fee: %d nanograms\n", estimatedFeeResponse.Result.SourceFees.InFwdFee)
		fmt.Printf("Storage Fee: %d nanograms\n", estimatedFeeResponse.Result.SourceFees.StorageFee)
		fmt.Printf("Gas Fee: %d nanograms\n", estimatedFeeResponse.Result.SourceFees.GasFee)
		fmt.Printf("Forward Fee: %d nanograms\n", estimatedFeeResponse.Result.SourceFees.FwdFee)
	} else {
		fmt.Printf("Error: %s\n", estimatedFeeResponse.Error)
	}
}
